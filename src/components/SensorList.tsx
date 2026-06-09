import { useStore } from '@/store'
import type { SensorInfo } from '@/types'
import { Droplets, Thermometer, Wind } from 'lucide-react'

function getTypeIcon(type: SensorInfo['type']) {
  switch (type) {
    case 'o2':
      return <Wind size={14} className="text-tech-blue" />
    case 'temperature':
      return <Thermometer size={14} className="text-tech-orange" />
    case 'humidity':
      return <Droplets size={14} className="text-tech-green" />
  }
}

function getStatusDot(status: string) {
  const color =
    status === 'online'
      ? 'bg-tech-green'
      : status === 'fault'
      ? 'bg-tech-red'
      : 'bg-gray-500'
  return <span className={`w-2 h-2 rounded-full ${color}`} />
}

export default function SensorList() {
  const sensors = useStore((s) => s.sensors)
  const setSelectedDetector = useStore((s) => s.setSelectedDetector)
  const detectors = useStore((s) => s.detectors)

  const handleSensorClick = (sensor: SensorInfo) => {
    const detector = detectors.find(
      (d) => d.partition_id === sensor.partition_id,
    )
    if (detector) {
      setSelectedDetector(detector)
    }
  }

  return (
    <div className="w-[280px] h-full bg-panel-light border-r border-panel-border flex flex-col shrink-0">
      <div className="px-3 py-2 border-b border-panel-border flex items-center justify-between">
        <span className="text-sm font-bold text-tech-blue">传感器列表</span>
        <span className="text-xs text-gray-400 bg-panel px-1.5 py-0.5 rounded">
          {sensors.length}
        </span>
      </div>
      <div className="flex-1 overflow-y-auto">
        {sensors.map((sensor) => (
          <div
            key={sensor.id}
            onClick={() => handleSensorClick(sensor)}
            className="flex items-center gap-2 px-3 py-2 border-b border-panel-border/50 hover:bg-panel-border/30 cursor-pointer transition-colors"
          >
            {getTypeIcon(sensor.type)}
            <div className="flex-1 min-w-0">
              <div className="text-xs text-gray-300 truncate">{sensor.id}</div>
            </div>
            <div className="text-right">
              <div className="text-xs font-mono">
                {sensor.latest_value.toFixed(1)}
                <span className="text-gray-500 ml-0.5">{sensor.latest_unit}</span>
              </div>
            </div>
            {getStatusDot(sensor.status)}
          </div>
        ))}
      </div>
    </div>
  )
}
