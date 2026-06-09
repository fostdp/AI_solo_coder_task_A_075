import { useStore } from '@/store'
import { Radio, WifiOff, AlertTriangle, Activity, Clock } from 'lucide-react'

export default function StatusBar() {
  const detectors = useStore((s) => s.detectors)
  const alarms = useStore((s) => s.alarms)

  const total = detectors.length
  const online = detectors.filter((d) => d.status === 'online').length
  const offline = detectors.filter((d) => d.status === 'offline').length
  const maxConc = detectors.length > 0
    ? Math.max(...detectors.map((d) => d.latest_concentration))
    : 0
  const activeAlarms = alarms.filter((a) => a.status === 'active').length

  return (
    <div className="h-8 bg-panel-light border-t border-panel-border flex items-center px-4 gap-6 text-xs shrink-0">
      <div className="flex items-center gap-1.5">
        <Radio size={12} className="text-tech-blue" />
        <span className="text-gray-400">检测器:</span>
        <span className="font-mono">{total}</span>
      </div>
      <div className="flex items-center gap-1.5">
        <Activity size={12} className="text-tech-green" />
        <span className="text-gray-400">在线:</span>
        <span className="font-mono text-tech-green">{online}</span>
      </div>
      <div className="flex items-center gap-1.5">
        <WifiOff size={12} className="text-gray-500" />
        <span className="text-gray-400">离线:</span>
        <span className="font-mono text-gray-500">{offline}</span>
      </div>
      <div className="flex items-center gap-1.5">
        <AlertTriangle size={12} className="text-tech-orange" />
        <span className="text-gray-400">最大浓度:</span>
        <span className={`font-mono ${maxConc >= 20 ? 'text-tech-red' : maxConc >= 10 ? 'text-tech-orange' : 'text-tech-green'}`}>
          {maxConc.toFixed(1)} %LEL
        </span>
      </div>
      <div className="flex items-center gap-1.5">
        <AlertTriangle size={12} className="text-tech-red" />
        <span className="text-gray-400">活动告警:</span>
        <span className={`font-mono ${activeAlarms > 0 ? 'text-tech-red' : 'text-tech-green'}`}>
          {activeAlarms}
        </span>
      </div>
      <div className="ml-auto flex items-center gap-1.5">
        <Clock size={12} className="text-tech-blue" />
        <span className="text-gray-400">刷新率:</span>
        <span className="font-mono text-tech-blue">1s</span>
      </div>
    </div>
  )
}
