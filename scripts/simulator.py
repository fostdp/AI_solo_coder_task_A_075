"""
Gas Leak Laser Monitoring System - Detector Simulator
Supports: 30km tunnel, 300 detectors, 1s interval, injectable leak sources, wind data injection
"""

import json
import math
import random
import time
import threading
import argparse
import sys
import os
from datetime import datetime, timezone
from http.client import HTTPConnection
from typing import Optional
from urllib.parse import urlparse

API_HOST = "localhost"
API_PORT = 8080
TUNNEL_LENGTH = 30000
NUM_DETECTORS = 300
NUM_O2_SENSORS = 20
NUM_TEMP_SENSORS = 15
NUM_HUMIDITY_SENSORS = 15
REPORT_INTERVAL = 1.0

BASE_LAT = 31.2304
BASE_LNG = 121.4737
LAT_RANGE = 0.0096
LNG_RANGE = 0.2163


def tunnel_distance_to_latlng(distance: float) -> tuple:
    t = distance / TUNNEL_LENGTH
    lat = BASE_LAT + t * LAT_RANGE + 0.002 * math.sin(2 * math.pi * distance / 10000)
    lng = BASE_LNG + t * LNG_RANGE + 0.003 * math.sin(2 * math.pi * distance / 8000)
    return lat, lng


class LeakSource:
    def __init__(self, distance: float, rate: float, duration: float = 0, leak_id: str = ""):
        self.distance = distance
        self.rate = rate
        self.duration = duration
        self.start_time = time.time()
        self.leak_id = leak_id or f"leak-{int(distance)}m-{time.time():.0f}"
        self.active = True

    def elapsed(self) -> float:
        return time.time() - self.start_time

    def is_expired(self) -> bool:
        if self.duration <= 0:
            return False
        return self.elapsed() > self.duration

    def concentration_at(self, detector_distance: float, wind_speed: float = 1.5, wind_dir: float = 90.0) -> float:
        dist = abs(detector_distance - self.distance)
        elapsed = self.elapsed()
        if self.duration > 0:
            ramp = min(1.0, elapsed / 5.0) * max(0.0, 1.0 - (elapsed - self.duration + 5.0) / 5.0) if elapsed > self.duration - 5.0 else min(1.0, elapsed / 5.0)
        else:
            ramp = min(1.0, elapsed / 5.0)
        wind_factor = math.exp(-dist / 500.0) * (1.0 + 0.3 * math.sin(elapsed * 0.5))
        sigma = 200.0 + wind_speed * 50.0
        contribution = self.rate * ramp * wind_factor * math.exp(-dist * dist / (2 * sigma * sigma))
        return max(0, contribution)


class WindField:
    def __init__(self):
        self.base_speed = 1.5
        self.base_direction = 90.0
        self.overrides: dict = {}
        self.lock = threading.Lock()

    def set_base(self, speed: float, direction: float):
        with self.lock:
            self.base_speed = speed
            self.base_direction = direction

    def set_partition_override(self, partition_id: str, speed: float, direction: float):
        with self.lock:
            self.overrides[partition_id] = {"speed": speed, "direction": direction}

    def clear_partition_override(self, partition_id: str):
        with self.lock:
            self.overrides.pop(partition_id, None)

    def get_wind(self, partition_id: str) -> tuple:
        with self.lock:
            if partition_id in self.overrides:
                o = self.overrides[partition_id]
                return o["speed"], o["direction"]
            return self.base_speed, self.base_direction


class LeakSimulator:
    def __init__(self, wind_field: WindField):
        self.active_leaks: list = []
        self.wind_field = wind_field
        self.leak_lock = threading.Lock()
        self.auto_leak = True
        self.max_leaks = 3

    def inject_leak(self, distance: float, rate: float, duration: float = 0) -> str:
        leak = LeakSource(distance, rate, duration)
        with self.leak_lock:
            self.active_leaks.append(leak)
        return leak.leak_id

    def remove_leak(self, leak_id: str) -> bool:
        with self.leak_lock:
            for i, l in enumerate(self.active_leaks):
                if l.leak_id == leak_id:
                    self.active_leaks.pop(i)
                    return True
        return False

    def maybe_trigger_leak(self):
        if not self.auto_leak:
            return
        if random.random() < 0.002 and len(self.active_leaks) < self.max_leaks:
            distance = random.uniform(1000, 29000)
            rate = random.uniform(5, 80)
            duration = random.uniform(30, 180)
            self.inject_leak(distance, rate, duration)

    def get_concentration(self, detector_distance: float, partition_distance: float) -> float:
        total = random.gauss(1.5, 0.5)
        wind_speed, wind_dir = self.wind_field.get_wind(f"P-{int(partition_distance / 1000) + 1:03d}")
        with self.leak_lock:
            for leak in self.active_leaks:
                if leak.active:
                    total += leak.concentration_at(detector_distance, wind_speed, wind_dir)
        return max(0, min(100, total))

    def cleanup(self):
        with self.leak_lock:
            for leak in self.active_leaks:
                if leak.is_expired():
                    leak.active = False
            self.active_leaks = [l for l in self.active_leaks if l.active]

    def get_active_leaks(self) -> list:
        with self.leak_lock:
            return [
                {
                    "id": l.leak_id,
                    "distance": l.distance,
                    "rate": l.rate,
                    "duration": l.duration,
                    "elapsed": l.elapsed(),
                    "active": l.active,
                }
                for l in self.active_leaks if l.active
            ]


def post_json(host: str, port: int, path: str, data: dict) -> Optional[dict]:
    try:
        conn = HTTPConnection(host, port, timeout=5)
        body = json.dumps(data)
        conn.request("POST", path, body=body, headers={"Content-Type": "application/json"})
        resp = conn.getresponse()
        result = resp.read().decode()
        conn.close()
        if resp.status == 200:
            return json.loads(result)
        return None
    except Exception:
        return None


def simulate_detectors(leak_sim: LeakSimulator, host: str, port: int):
    while True:
        leak_sim.maybe_trigger_leak()
        leak_sim.cleanup()

        for i in range(1, NUM_DETECTORS + 1):
            distance = i * 100.0
            partition_distance = distance
            concentration = leak_sim.get_concentration(distance, partition_distance)
            detector_id = f"D-{i:03d}"
            status = "online" if random.random() > 0.001 else "fault"
            data = {
                "detector_id": detector_id,
                "concentration": round(concentration, 2),
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": status,
            }
            post_json(host, port, "/api/detectors/data", data)

        time.sleep(REPORT_INTERVAL)


def simulate_sensors(host: str, port: int):
    while True:
        for i in range(1, NUM_O2_SENSORS + 1):
            sensor_id = f"S-O2-{i:03d}"
            value = round(random.gauss(20.9, 0.3), 1)
            data = {
                "sensor_id": sensor_id,
                "type": "o2",
                "value": max(18.0, min(23.0, value)),
                "unit": "%",
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": "online" if random.random() > 0.002 else "fault",
            }
            post_json(host, port, "/api/sensors/data", data)

        for i in range(1, NUM_TEMP_SENSORS + 1):
            sensor_id = f"S-T-{i:03d}"
            value = round(random.gauss(22.0, 2.0), 1)
            data = {
                "sensor_id": sensor_id,
                "type": "temperature",
                "value": max(10.0, min(40.0, value)),
                "unit": "C",
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": "online" if random.random() > 0.002 else "fault",
            }
            post_json(host, port, "/api/sensors/data", data)

        for i in range(1, NUM_HUMIDITY_SENSORS + 1):
            sensor_id = f"S-H-{i:03d}"
            value = round(random.gauss(65.0, 5.0), 1)
            data = {
                "sensor_id": sensor_id,
                "type": "humidity",
                "value": max(30.0, min(95.0, value)),
                "unit": "%RH",
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": "online" if random.random() > 0.002 else "fault",
            }
            post_json(host, port, "/api/sensors/data", data)

        time.sleep(REPORT_INTERVAL)


def simulate_wind(wind_field: WindField, host: str, port: int):
    while True:
        for p in range(1, 31):
            partition_id = f"P-{p:03d}"
            speed, direction = wind_field.get_wind(partition_id)
            jitter_speed = max(0, round(speed + random.gauss(0, 0.2), 2))
            jitter_dir = round((direction + random.gauss(0, 5)) % 360, 1)
            data = {
                "partition_id": partition_id,
                "speed": jitter_speed,
                "direction": jitter_dir,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
            post_json(host, port, "/api/wind/data", data)
        time.sleep(5)


def interactive_cli(leak_sim: LeakSimulator, wind_field: WindField):
    print("\nInteractive Commands:")
    print("  leak <distance_m> <rate_%LEL> [duration_s]  - Inject leak source")
    print("  leak-remove <leak_id>                        - Remove leak source")
    print("  leaks                                        - List active leaks")
    print("  wind <speed> <direction>                     - Set base wind")
    print("  wind-partition <partition> <speed> <dir>     - Set partition wind override")
    print("  auto-leak [on|off]                           - Toggle auto leak generation")
    print("  quit                                         - Exit")
    print()

    while True:
        try:
            cmd = input("> ").strip()
            if not cmd:
                continue
            parts = cmd.split()
            action = parts[0].lower()

            if action == "quit":
                print("Stopping simulator...")
                sys.exit(0)

            elif action == "leak":
                if len(parts) < 3:
                    print("Usage: leak <distance_m> <rate_%LEL> [duration_s]")
                    continue
                distance = float(parts[1])
                rate = float(parts[2])
                duration = float(parts[3]) if len(parts) > 3 else 0
                leak_id = leak_sim.inject_leak(distance, rate, duration)
                lat, lng = tunnel_distance_to_latlng(distance)
                print(f"Injected leak: id={leak_id}, dist={distance}m, rate={rate}%LEL, "
                      f"duration={'permanent' if duration == 0 else f'{duration}s'}, "
                      f"lat={lat:.6f}, lng={lng:.6f}")

            elif action == "leak-remove":
                if len(parts) < 2:
                    print("Usage: leak-remove <leak_id>")
                    continue
                if leak_sim.remove_leak(parts[1]):
                    print(f"Removed leak: {parts[1]}")
                else:
                    print(f"Leak not found: {parts[1]}")

            elif action == "leaks":
                leaks = leak_sim.get_active_leaks()
                if not leaks:
                    print("No active leaks")
                for l in leaks:
                    dur_str = f"{l['duration']:.0f}s" if l['duration'] > 0 else "permanent"
                    remaining = (l['duration'] - l['elapsed']) if l['duration'] > 0 else float('inf')
                    rem_str = f"{remaining:.0f}s" if remaining != float('inf') else "permanent"
                    print(f"  {l['id']}: dist={l['distance']:.0f}m, rate={l['rate']:.1f}%LEL, "
                          f"duration={dur_str}, remaining={rem_str}")

            elif action == "wind":
                if len(parts) < 3:
                    print("Usage: wind <speed> <direction>")
                    continue
                speed = float(parts[1])
                direction = float(parts[2])
                wind_field.set_base(speed, direction)
                print(f"Base wind set: speed={speed} m/s, direction={direction} deg")

            elif action == "wind-partition":
                if len(parts) < 4:
                    print("Usage: wind-partition <partition_id> <speed> <direction>")
                    continue
                partition_id = parts[1]
                speed = float(parts[2])
                direction = float(parts[3])
                wind_field.set_partition_override(partition_id, speed, direction)
                print(f"Wind override set for {partition_id}: speed={speed} m/s, direction={direction} deg")

            elif action == "auto-leak":
                if len(parts) > 1:
                    leak_sim.auto_leak = parts[1].lower() in ("on", "true", "1")
                else:
                    leak_sim.auto_leak = not leak_sim.auto_leak
                print(f"Auto leak generation: {'ON' if leak_sim.auto_leak else 'OFF'}")

            else:
                print(f"Unknown command: {action}")

        except EOFError:
            break
        except KeyboardInterrupt:
            break
        except Exception as e:
            print(f"Error: {e}")


def main():
    global API_HOST, API_PORT, TUNNEL_LENGTH, NUM_DETECTORS, REPORT_INTERVAL
    global NUM_O2_SENSORS, NUM_TEMP_SENSORS, NUM_HUMIDITY_SENSORS

    parser = argparse.ArgumentParser(description="Gas Leak Detector Simulator")
    parser.add_argument("--host", default=os.environ.get("API_HOST", "localhost"), help="API host")
    parser.add_argument("--port", type=int, default=int(os.environ.get("API_PORT", "8080")), help="API port")
    parser.add_argument("--tunnel-length", type=float, default=float(os.environ.get("TUNNEL_LENGTH", "30000")), help="Tunnel length in meters")
    parser.add_argument("--detectors", type=int, default=int(os.environ.get("NUM_DETECTORS", "300")), help="Number of detectors")
    parser.add_argument("--interval", type=float, default=float(os.environ.get("REPORT_INTERVAL", "1.0")), help="Report interval in seconds")
    parser.add_argument("--o2-sensors", type=int, default=int(os.environ.get("NUM_O2_SENSORS", "20")), help="O2 sensor count")
    parser.add_argument("--temp-sensors", type=int, default=int(os.environ.get("NUM_TEMP_SENSORS", "15")), help="Temperature sensor count")
    parser.add_argument("--humidity-sensors", type=int, default=int(os.environ.get("NUM_HUMIDITY_SENSORS", "15")), help="Humidity sensor count")
    parser.add_argument("--no-auto-leak", action="store_true", help="Disable automatic leak generation")
    parser.add_argument("--leak-distance", type=float, default=float(os.environ.get("LEAK_DISTANCE", "0")) or None, help="Inject initial leak at distance (m)")
    parser.add_argument("--leak-rate", type=float, default=float(os.environ.get("LEAK_RATE", "0")) or None, help="Initial leak rate (%LEL)")
    parser.add_argument("--leak-duration", type=float, default=float(os.environ.get("LEAK_DURATION", "0")), help="Initial leak duration (0=permanent)")
    parser.add_argument("--wind-speed", type=float, default=float(os.environ.get("WIND_SPEED", "0")) or None, help="Base wind speed (m/s)")
    parser.add_argument("--wind-direction", type=float, default=float(os.environ.get("WIND_DIRECTION", "0")) or None, help="Base wind direction (degrees)")
    parser.add_argument("--interactive", action="store_true", help="Enable interactive CLI")

    args = parser.parse_args()

    API_HOST = args.host
    API_PORT = args.port
    TUNNEL_LENGTH = args.tunnel_length
    NUM_DETECTORS = args.detectors
    REPORT_INTERVAL = args.interval
    NUM_O2_SENSORS = args.o2_sensors
    NUM_TEMP_SENSORS = args.temp_sensors
    NUM_HUMIDITY_SENSORS = args.humidity_sensors

    wind_field = WindField()
    if args.wind_speed is not None:
        wind_field.set_base(args.wind_speed, args.wind_direction if args.wind_direction is not None else 90.0)

    leak_sim = LeakSimulator(wind_field)
    if args.no_auto_leak:
        leak_sim.auto_leak = False

    if args.leak_distance is not None and args.leak_rate is not None:
        leak_id = leak_sim.inject_leak(args.leak_distance, args.leak_rate, args.leak_duration)
    else:
        leak_id = None

    print("=" * 60)
    print("  Gas Leak Laser Monitoring System - Simulator")
    print("=" * 60)
    print(f"  Tunnel Length:    {TUNNEL_LENGTH}m")
    print(f"  Detectors:        {NUM_DETECTORS}")
    print(f"  O2 Sensors:       {NUM_O2_SENSORS}")
    print(f"  Temp Sensors:     {NUM_TEMP_SENSORS}")
    print(f"  Humidity Sensors: {NUM_HUMIDITY_SENSORS}")
    print(f"  Report Interval:  {REPORT_INTERVAL}s")
    print(f"  Target API:       http://{API_HOST}:{API_PORT}")
    print(f"  Auto Leak:        {'ON' if leak_sim.auto_leak else 'OFF'}")
    print(f"  Base Wind:        {wind_field.base_speed:.1f} m/s @ {wind_field.base_direction:.0f} deg")
    if leak_id:
        print(f"  Initial Leak:     dist={args.leak_distance}m, rate={args.leak_rate}%LEL")
    print("=" * 60)

    t1 = threading.Thread(target=simulate_detectors, args=(leak_sim, API_HOST, API_PORT), daemon=True)
    t2 = threading.Thread(target=simulate_sensors, args=(API_HOST, API_PORT), daemon=True)
    t3 = threading.Thread(target=simulate_wind, args=(wind_field, API_HOST, API_PORT), daemon=True)

    t1.start()
    t2.start()
    t3.start()

    if args.interactive:
        interactive_cli(leak_sim, wind_field)
    else:
        try:
            while True:
                active = leak_sim.get_active_leaks()
                if active:
                    for leak in active:
                        dist_km = leak["distance"] / 1000
                        print(f"  WARNING Leak: pos={dist_km:.1f}km, "
                              f"rate={leak['rate']:.1f}%LEL, "
                              f"id={leak['id']}")
                time.sleep(5)
        except KeyboardInterrupt:
            print("\nSimulator stopped")


if __name__ == "__main__":
    main()
