"""
智慧城市地下综合管廊燃气泄漏激光监测系统 - 激光检测器模拟器
模拟300台激光甲烷检测器 + 50台环境传感器每秒上报数据
"""

import json
import math
import random
import time
import threading
from datetime import datetime, timezone
from http.client import HTTPConnection
from typing import Optional

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


class LeakSimulator:
    def __init__(self):
        self.active_leaks: list = []
        self.leak_lock = threading.Lock()

    def maybe_trigger_leak(self):
        if random.random() < 0.002 and len(self.active_leaks) < 3:
            distance = random.uniform(1000, 29000)
            rate = random.uniform(5, 80)
            duration = random.uniform(30, 180)
            self.active_leaks.append({
                "distance": distance,
                "rate": rate,
                "start_time": time.time(),
                "duration": duration,
            })

    def get_concentration(self, detector_distance: float) -> float:
        total = random.gauss(1.5, 0.5)
        with self.leak_lock:
            for leak in self.active_leaks:
                dist = abs(detector_distance - leak["distance"])
                elapsed = time.time() - leak["start_time"]
                if elapsed > leak["duration"]:
                    continue
                wind_factor = math.exp(-dist / 500.0) * (1.0 + 0.3 * math.sin(elapsed * 0.5))
                contribution = leak["rate"] * wind_factor * math.exp(-dist * dist / (2 * 200 * 200))
                total += max(0, contribution)
        return max(0, min(100, total))

    def cleanup(self):
        with self.leak_lock:
            now = time.time()
            self.active_leaks = [
                l for l in self.active_leaks
                if now - l["start_time"] < l["duration"]
            ]


def post_json(path: str, data: dict) -> Optional[dict]:
    try:
        conn = HTTPConnection(API_HOST, API_PORT, timeout=5)
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


def simulate_detectors(leak_sim: LeakSimulator):
    while True:
        leak_sim.maybe_trigger_leak()
        leak_sim.cleanup()

        batch = []
        for i in range(1, NUM_DETECTORS + 1):
            distance = i * 100.0
            concentration = leak_sim.get_concentration(distance)
            detector_id = f"D-{i:03d}"
            status = "online" if random.random() > 0.001 else "fault"
            batch.append({
                "detector_id": detector_id,
                "concentration": round(concentration, 2),
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": status,
            })

        for data in batch:
            post_json("/api/detectors/data", data)

        time.sleep(REPORT_INTERVAL)


def simulate_sensors():
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
            post_json("/api/sensors/data", data)

        for i in range(1, NUM_TEMP_SENSORS + 1):
            sensor_id = f"S-T-{i:03d}"
            value = round(random.gauss(22.0, 2.0), 1)
            data = {
                "sensor_id": sensor_id,
                "type": "temperature",
                "value": max(10.0, min(40.0, value)),
                "unit": "°C",
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "status": "online" if random.random() > 0.002 else "fault",
            }
            post_json("/api/sensors/data", data)

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
            post_json("/api/sensors/data", data)

        time.sleep(REPORT_INTERVAL)


def simulate_wind():
    while True:
        for p in range(1, 31):
            partition_id = f"P-{p:03d}"
            speed = round(random.gauss(1.5, 0.5), 2)
            direction = round(random.gauss(90, 30) % 360, 1)
            data = {
                "partition_id": partition_id,
                "speed": max(0, speed),
                "direction": direction,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
            post_json("/api/wind/data", data)
        time.sleep(5)


def main():
    print("=" * 60)
    print("  管廊燃气泄漏激光监测系统 - 模拟器启动")
    print("=" * 60)
    print(f"  检测器数量: {NUM_DETECTORS}")
    print(f"  O2传感器: {NUM_O2_SENSORS}")
    print(f"  温度传感器: {NUM_TEMP_SENSORS}")
    print(f"  湿度传感器: {NUM_HUMIDITY_SENSORS}")
    print(f"  上报间隔: {REPORT_INTERVAL}s")
    print(f"  目标API: http://{API_HOST}:{API_PORT}")
    print("=" * 60)
    print("  模拟泄漏: 随机触发，浓度基于高斯扩散模型")
    print("  按 Ctrl+C 停止")
    print("=" * 60)

    leak_sim = LeakSimulator()

    t1 = threading.Thread(target=simulate_detectors, args=(leak_sim,), daemon=True)
    t2 = threading.Thread(target=simulate_sensors, daemon=True)
    t3 = threading.Thread(target=simulate_wind, daemon=True)

    t1.start()
    t2.start()
    t3.start()

    try:
        while True:
            active = len(leak_sim.active_leaks)
            if active > 0:
                for leak in leak_sim.active_leaks:
                    dist_km = leak["distance"] / 1000
                    print(f"  ⚠ 泄漏事件: 位置 {dist_km:.1f}km, "
                          f"泄漏速率 {leak['rate']:.1f}%LEL, "
                          f"剩余 {leak['duration'] - (time.time() - leak['start_time']):.0f}s")
            time.sleep(5)
    except KeyboardInterrupt:
        print("\n模拟器已停止")


if __name__ == "__main__":
    main()
