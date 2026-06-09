import { useStore } from '@/store'
import { api } from '@/utils/api'
import { Bell, Check } from 'lucide-react'
import type { Alarm } from '@/types'

const LEVEL_COLORS: Record<number, string> = {
  1: 'border-l-tech-yellow',
  2: 'border-l-tech-orange',
  3: 'border-l-tech-red',
}

const LEVEL_BG: Record<number, string> = {
  1: 'bg-tech-yellow/15 text-tech-yellow',
  2: 'bg-tech-orange/15 text-tech-orange',
  3: 'bg-tech-red/15 text-tech-red',
}

const LEVEL_LABELS: Record<number, string> = {
  1: 'L1',
  2: 'L2',
  3: 'L3',
}

const STATUS_LABELS: Record<string, string> = {
  active: '活动',
  acknowledged: '已确认',
  resolved: '已解除',
}

function formatTime(iso: string) {
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`
}

export default function AlarmPanel() {
  const alarms = useStore((s) => s.alarms)
  const updateAlarm = useStore((s) => s.updateAlarm)

  const activeAlarms = alarms.filter((a) => a.status !== 'resolved')
  const alarmCount = activeAlarms.length

  const handleAcknowledge = async (alarm: Alarm) => {
    try {
      const updated = await api.acknowledgeAlarm(alarm.id)
      updateAlarm(alarm.id, updated)
    } catch {}
  }

  const sortedAlarms = [...alarms].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  )

  return (
    <div className="w-[320px] h-full bg-panel-light border-l border-panel-border flex flex-col shrink-0">
      <div className="px-3 py-2 border-b border-panel-border flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Bell size={14} className="text-tech-red" />
          <span className="text-sm font-bold text-tech-blue">告警面板</span>
        </div>
        {alarmCount > 0 && (
          <span className="text-xs bg-tech-red/20 text-tech-red px-1.5 py-0.5 rounded font-mono">
            {alarmCount}
          </span>
        )}
      </div>
      <div className="flex-1 overflow-y-auto">
        {sortedAlarms.map((alarm) => (
          <div
            key={alarm.id}
            className={`border-l-4 ${LEVEL_COLORS[alarm.level]} px-3 py-2 border-b border-panel-border/50 ${
              alarm.level === 3 && alarm.status === 'active' ? 'animate-blink' : ''
            }`}
          >
            <div className="flex items-center justify-between mb-1">
              <div className="flex items-center gap-1.5">
                <span className={`text-[10px] px-1.5 py-0.5 rounded font-bold ${LEVEL_BG[alarm.level]}`}>
                  {LEVEL_LABELS[alarm.level]}
                </span>
                <span className="text-xs text-gray-300">{alarm.detector_id}</span>
              </div>
              <span className={`text-[10px] px-1.5 py-0.5 rounded ${
                alarm.status === 'active'
                  ? 'bg-tech-red/15 text-tech-red'
                  : alarm.status === 'acknowledged'
                  ? 'bg-tech-yellow/15 text-tech-yellow'
                  : 'bg-tech-green/15 text-tech-green'
              }`}>
                {STATUS_LABELS[alarm.status]}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-mono text-tech-red">
                {alarm.concentration.toFixed(1)} %LEL
              </span>
              <span className="text-[10px] text-gray-500">
                {formatTime(alarm.created_at)}
              </span>
            </div>
            {alarm.status === 'active' && (
              <button
                onClick={() => handleAcknowledge(alarm)}
                className="mt-1.5 flex items-center gap-1 text-[10px] text-tech-blue hover:text-white bg-tech-blue/10 hover:bg-tech-blue/20 px-2 py-0.5 rounded transition-colors"
              >
                <Check size={10} />
                确认
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
