import { useEffect, useState, useCallback } from 'react'
import { useStore } from '@/store'
import { api } from '@/utils/api'
import type { Alarm } from '@/types'
import { Filter, CheckCircle, XCircle, ChevronLeft, ChevronRight } from 'lucide-react'

const LEVEL_COLORS: Record<number, string> = {
  1: 'text-tech-yellow',
  2: 'text-tech-orange',
  3: 'text-tech-red',
}

const LEVEL_BG: Record<number, string> = {
  1: 'bg-tech-yellow/15 text-tech-yellow',
  2: 'bg-tech-orange/15 text-tech-orange',
  3: 'bg-tech-red/15 text-tech-red',
}

const STATUS_LABELS: Record<string, string> = {
  active: '活动',
  acknowledged: '已确认',
  resolved: '已解除',
}

const STATUS_BG: Record<string, string> = {
  active: 'bg-tech-red/15 text-tech-red',
  acknowledged: 'bg-tech-yellow/15 text-tech-yellow',
  resolved: 'bg-tech-green/15 text-tech-green',
}

const PAGE_SIZE = 20

export default function AlarmPage() {
  const storeAlarms = useStore((s) => s.alarms)
  const updateAlarm = useStore((s) => s.updateAlarm)
  const [levelFilter, setLevelFilter] = useState<string>('all')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [page, setPage] = useState(1)

  const filteredAlarms = storeAlarms.filter((a) => {
    if (levelFilter !== 'all' && a.level !== Number(levelFilter)) return false
    if (statusFilter !== 'all' && a.status !== statusFilter) return false
    if (startDate && new Date(a.created_at) < new Date(startDate)) return false
    if (endDate && new Date(a.created_at) > new Date(endDate + 'T23:59:59')) return false
    return true
  })

  const totalPages = Math.max(1, Math.ceil(filteredAlarms.length / PAGE_SIZE))
  const paginatedAlarms = filteredAlarms.slice(
    (page - 1) * PAGE_SIZE,
    page * PAGE_SIZE,
  )

  const handleAcknowledge = useCallback(
    async (alarm: Alarm) => {
      try {
        const updated = await api.acknowledgeAlarm(alarm.id)
        updateAlarm(alarm.id, updated)
      } catch {}
    },
    [updateAlarm],
  )

  const handleResolve = useCallback(
    async (alarm: Alarm) => {
      try {
        const updated = await api.resolveAlarm(alarm.id)
        updateAlarm(alarm.id, updated)
      } catch {}
    },
    [updateAlarm],
  )

  useEffect(() => {
    setPage(1)
  }, [levelFilter, statusFilter, startDate, endDate])

  return (
    <div className="h-full flex flex-col bg-panel">
      <div className="px-4 py-3 border-b border-panel-border flex items-center gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <Filter size={14} className="text-tech-blue" />
          <span className="text-xs text-gray-400">级别:</span>
          <select
            value={levelFilter}
            onChange={(e) => setLevelFilter(e.target.value)}
            className="bg-panel-light border border-panel-border text-sm rounded px-2 py-1 text-gray-200 focus:outline-none focus:border-tech-blue"
          >
            <option value="all">全部</option>
            <option value="1">L1</option>
            <option value="2">L2</option>
            <option value="3">L3</option>
          </select>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-400">状态:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="bg-panel-light border border-panel-border text-sm rounded px-2 py-1 text-gray-200 focus:outline-none focus:border-tech-blue"
          >
            <option value="all">全部</option>
            <option value="active">活动</option>
            <option value="acknowledged">已确认</option>
            <option value="resolved">已解除</option>
          </select>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-400">起始:</span>
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="bg-panel-light border border-panel-border text-sm rounded px-2 py-1 text-gray-200 focus:outline-none focus:border-tech-blue"
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-400">截止:</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="bg-panel-light border border-panel-border text-sm rounded px-2 py-1 text-gray-200 focus:outline-none focus:border-tech-blue"
          />
        </div>
        <div className="text-xs text-gray-400 ml-auto">
          共 {filteredAlarms.length} 条告警
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-panel-light z-10">
            <tr className="border-b border-panel-border">
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">级别</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">检测器ID</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">浓度</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">阈值</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">消息</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">状态</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">时间</th>
              <th className="text-left px-3 py-2 text-xs text-gray-400 font-normal">操作</th>
            </tr>
          </thead>
          <tbody>
            {paginatedAlarms.map((alarm) => (
              <tr
                key={alarm.id}
                className="border-b border-panel-border/50 hover:bg-panel-light/50"
              >
                <td className="px-3 py-2">
                  <span className={`text-xs px-1.5 py-0.5 rounded font-bold ${LEVEL_BG[alarm.level]}`}>
                    L{alarm.level}
                  </span>
                </td>
                <td className="px-3 py-2 font-mono text-xs">{alarm.detector_id}</td>
                <td className={`px-3 py-2 font-mono text-xs ${LEVEL_COLORS[alarm.level]}`}>
                  {alarm.concentration.toFixed(1)} %LEL
                </td>
                <td className="px-3 py-2 font-mono text-xs text-gray-400">
                  {alarm.threshold.toFixed(1)} %LEL
                </td>
                <td className="px-3 py-2 text-xs text-gray-300 max-w-[200px] truncate">
                  {alarm.message}
                </td>
                <td className="px-3 py-2">
                  <span className={`text-[10px] px-1.5 py-0.5 rounded ${STATUS_BG[alarm.status]}`}>
                    {STATUS_LABELS[alarm.status]}
                  </span>
                </td>
                <td className="px-3 py-2 text-xs text-gray-400 font-mono">
                  {new Date(alarm.created_at).toLocaleString('zh-CN')}
                </td>
                <td className="px-3 py-2">
                  <div className="flex gap-1">
                    {alarm.status === 'active' && (
                      <button
                        onClick={() => handleAcknowledge(alarm)}
                        className="flex items-center gap-1 text-[10px] text-tech-blue bg-tech-blue/10 hover:bg-tech-blue/20 px-2 py-0.5 rounded transition-colors"
                      >
                        <CheckCircle size={10} />
                        确认
                      </button>
                    )}
                    {alarm.status === 'acknowledged' && (
                      <button
                        onClick={() => handleResolve(alarm)}
                        className="flex items-center gap-1 text-[10px] text-tech-green bg-tech-green/10 hover:bg-tech-green/20 px-2 py-0.5 rounded transition-colors"
                      >
                        <XCircle size={10} />
                        解除
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="h-10 border-t border-panel-border flex items-center justify-center gap-4 shrink-0">
        <button
          onClick={() => setPage((p) => Math.max(1, p - 1))}
          disabled={page <= 1}
          className="text-gray-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronLeft size={18} />
        </button>
        <span className="text-xs text-gray-400">
          {page} / {totalPages}
        </span>
        <button
          onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
          className="text-gray-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <ChevronRight size={18} />
        </button>
      </div>
    </div>
  )
}
