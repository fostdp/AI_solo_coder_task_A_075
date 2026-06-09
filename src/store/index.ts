import { create } from 'zustand'
import type { DetectorInfo, SensorInfo, Alarm, LeakSourceResult, Partition, WindData, WSMessage } from '@/types'

interface AppState {
  detectors: DetectorInfo[]
  sensors: SensorInfo[]
  alarms: Alarm[]
  leakSource: LeakSourceResult | null
  partitions: Partition[]
  windData: WindData[]
  selectedDetector: DetectorInfo | null
  wsConnected: boolean

  setDetectors: (detectors: DetectorInfo[]) => void
  updateDetector: (id: string, concentration: number, timestamp: string) => void
  setSensors: (sensors: SensorInfo[]) => void
  updateSensor: (id: string, value: number, timestamp: string) => void
  addAlarm: (alarm: Alarm) => void
  updateAlarm: (id: string, updates: Partial<Alarm>) => void
  setAlarms: (alarms: Alarm[]) => void
  setLeakSource: (result: LeakSourceResult | null) => void
  setPartitions: (partitions: Partition[]) => void
  updatePartitionState: (id: string, field: 'valve_state' | 'fan_state', value: string) => void
  setWindData: (data: WindData[]) => void
  setSelectedDetector: (detector: DetectorInfo | null) => void
  setWsConnected: (connected: boolean) => void
  handleWSMessage: (msg: WSMessage) => void
}

export const useStore = create<AppState>((set, get) => ({
  detectors: [],
  sensors: [],
  alarms: [],
  leakSource: null,
  partitions: [],
  windData: [],
  selectedDetector: null,
  wsConnected: false,

  setDetectors: (detectors) => set({ detectors }),
  updateDetector: (id, concentration, timestamp) =>
    set((state) => ({
      detectors: state.detectors.map((d) =>
        d.id === id ? { ...d, latest_concentration: concentration, latest_timestamp: timestamp } : d
      ),
    })),
  setSensors: (sensors) => set({ sensors }),
  updateSensor: (id, value, timestamp) =>
    set((state) => ({
      sensors: state.sensors.map((s) =>
        s.id === id ? { ...s, latest_value: value } : s
      ),
    })),
  addAlarm: (alarm) => set((state) => ({ alarms: [alarm, ...state.alarms] })),
  updateAlarm: (id, updates) =>
    set((state) => ({
      alarms: state.alarms.map((a) => (a.id === id ? { ...a, ...updates } : a)),
    })),
  setAlarms: (alarms) => set({ alarms }),
  setLeakSource: (result) => set({ leakSource: result }),
  setPartitions: (partitions) => set({ partitions }),
  updatePartitionState: (id, field, value) =>
    set((state) => ({
      partitions: state.partitions.map((p) =>
        p.id === id ? { ...p, [field]: value } : p
      ),
    })),
  setWindData: (data) => set({ windData: data }),
  setSelectedDetector: (detector) => set({ selectedDetector: detector }),
  setWsConnected: (connected) => set({ wsConnected: connected }),

  handleWSMessage: (msg) => {
    switch (msg.type) {
      case 'detector_update': {
        const p = msg.payload as { detector_id: string; concentration: number; timestamp: string; status: string }
        get().updateDetector(p.detector_id, p.concentration, p.timestamp)
        break
      }
      case 'sensor_update': {
        const p = msg.payload as { sensor_id: string; value: number; timestamp: string }
        get().updateSensor(p.sensor_id, p.value, p.timestamp)
        break
      }
      case 'alarm': {
        const p = msg.payload as Alarm
        get().addAlarm(p)
        break
      }
      case 'leak_update': {
        const p = msg.payload as LeakSourceResult
        get().setLeakSource(p)
        break
      }
      case 'control_update': {
        const p = msg.payload as { partition_id: string; device_type: string; new_state: string }
        if (p.device_type === 'valve') {
          get().updatePartitionState(p.partition_id, 'valve_state', p.new_state)
        } else if (p.device_type === 'fan') {
          get().updatePartitionState(p.partition_id, 'fan_state', p.new_state)
        }
        break
      }
      case 'wind_update': {
        const p = msg.payload as WindData
        set((state) => ({
          windData: [...state.windData.filter(w => w.partition_id !== p.partition_id), p],
        }))
        break
      }
    }
  },
}))
