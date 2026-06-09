export interface DetectorData {
  detector_id: string
  concentration: number
  timestamp: string
  status: 'online' | 'offline' | 'fault'
}

export interface DetectorInfo {
  id: string
  name: string
  latitude: number
  longitude: number
  distance_along_tunnel: number
  partition_id: string
  status: string
  latest_concentration: number
  latest_timestamp: string
}

export interface SensorData {
  sensor_id: string
  type: 'o2' | 'temperature' | 'humidity'
  value: number
  unit: string
  timestamp: string
  status: 'online' | 'offline' | 'fault'
}

export interface SensorInfo {
  id: string
  name: string
  type: 'o2' | 'temperature' | 'humidity'
  latitude: number
  longitude: number
  partition_id: string
  status: string
  latest_value: number
  latest_unit: string
}

export interface HistoryPoint {
  time: string
  avg: number
  max: number
  min: number
}

export interface DetectorHistory {
  detector_id: string
  points: HistoryPoint[]
}

export interface DetectorHealth {
  detector_id: string
  status: 'online' | 'offline' | 'fault'
  uptime_hours: number
  last_calibration: string
  battery_level: number
  signal_strength: number
  error_count_24h: number
}

export interface Alarm {
  id: string
  level: 1 | 2 | 3
  detector_id: string
  concentration: number
  threshold: number
  message: string
  status: 'active' | 'acknowledged' | 'resolved'
  created_at: string
  updated_at: string
}

export interface LeakSourceResult {
  source_position: { lat: number; lng: number; distance: number }
  leak_rate: number
  confidence: number
  diffusion_radius: number
  method: 'pso' | 'bayesian'
  timestamp: string
}

export interface ControlCommand {
  device_id: string
  device_type: 'valve' | 'fan' | 'notification'
  action: 'open' | 'close' | 'start' | 'stop' | 'send'
  partition_id?: string
  message?: string
}

export interface ControlResult {
  success: boolean
  device_id: string
  new_state: string
  timestamp: string
}

export interface Partition {
  id: string
  name: string
  start_distance: number
  end_distance: number
  valve_id: string
  fan_id: string
  valve_state: 'open' | 'closed'
  fan_state: 'running' | 'stopped'
}

export interface WindData {
  partition_id: string
  speed: number
  direction: number
  timestamp: string
}

export interface WSMessage {
  type: 'detector_update' | 'sensor_update' | 'alarm' | 'leak_update' | 'control_update' | 'wind_update'
  payload: unknown
}
