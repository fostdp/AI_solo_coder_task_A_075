import { useEffect, useRef, useCallback, useState } from 'react'
import { useStore } from '@/store'
import { api } from '@/utils/api'
import { GasPanel } from '@/utils/gas_panel'
import type { DetectorHistory, DetectorHealth } from '@/types'
import { X, Activity, Signal, Wrench, AlertTriangle, Crosshair } from 'lucide-react'

export default function DetectorPopup() {
  const selectedDetector = useStore((s) => s.selectedDetector)
  const setSelectedDetector = useStore((s) => s.setSelectedDetector)
  const setLeakSource = useStore((s) => s.setLeakSource)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const gasPanelRef = useRef<GasPanel | null>(null)
  const [history, setHistory] = useState<DetectorHistory | null>(null)
  const [health, setHealth] = useState<DetectorHealth | null>(null)

  useEffect(() => {
    if (canvasRef.current) {
      gasPanelRef.current = new GasPanel(canvasRef.current)
    }
  }, [])

  useEffect(() => {
    if (!selectedDetector) return
    const now = new Date()
    const start = new Date(now.getTime() - 60 * 60 * 1000).toISOString()
    api.getDetectorHistory(selectedDetector.id, start, now.toISOString()).then(setHistory).catch(() => {})
    api.getDetectorHealth(selectedDetector.id).then(setHealth).catch(() => {})
  }, [selectedDetector])

  useEffect(() => {
    if (!history || !gasPanelRef.current) return
    gasPanelRef.current.drawTrendChart(history.points)
  }, [history])

  const handleLocate = useCallback(async () => {
    if (!selectedDetector) return
    try {
      const result = await api.locateLeak({
        detector_ids: [selectedDetector.id],
        concentrations: [selectedDetector.latest_concentration],
        wind_speed: 0,
        wind_direction: 0,
      })
      setLeakSource(result)
    } catch {}
  }, [selectedDetector, setLeakSource])

  if (!selectedDetector) return null

  const concColor = GasPanel.getConcentrationColor(selectedDetector.latest_concentration)

  return (
    <div className="absolute top-4 right-4 z-[1000] w-[320px] bg-panel-light border border-panel-border rounded-lg shadow-xl">
      <div className="flex items-center justify-between px-3 py-2 border-b border-panel-border">
        <div className="flex items-center gap-2">
          <Activity size={14} className="text-tech-blue" />
          <span className="text-sm font-bold text-tech-blue">{selectedDetector.id}</span>
        </div>
        <button onClick={() => setSelectedDetector(null)} className="text-gray-400 hover:text-white">
          <X size={16} />
        </button>
      </div>
      <div className="p-3 space-y-3">
        <div className="flex justify-between items-center">
          <span className="text-xs text-gray-400">名称</span>
          <span className="text-sm">{selectedDetector.name}</span>
        </div>
        <div className="flex justify-between items-center">
          <span className="text-xs text-gray-400">浓度</span>
          <span className="text-lg font-bold font-mono" style={{ color: concColor }}>
            {GasPanel.formatConcentration(selectedDetector.latest_concentration)} %LEL
          </span>
        </div>
        <div className="flex justify-between items-center">
          <span className="text-xs text-gray-400">状态</span>
          <span className={`text-xs px-2 py-0.5 rounded ${GasPanel.getStatusClass(selectedDetector.status)}`}>
            {GasPanel.getStatusLabel(selectedDetector.status)}
          </span>
        </div>
        <div>
          <div className="text-xs text-gray-400 mb-1">浓度趋势 (1h)</div>
          <canvas ref={canvasRef} width={280} height={120} className="w-full rounded border border-panel-border" />
        </div>
        {health && (
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex items-center gap-1.5">
              <Signal size={12} className="text-tech-blue" />
              <span className="text-gray-400">信号:</span>
              <span>{health.signal_strength}%</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Wrench size={12} className="text-tech-blue" />
              <span className="text-gray-400">校准:</span>
              <span>{new Date(health.last_calibration).toLocaleDateString('zh-CN')}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Activity size={12} className="text-tech-blue" />
              <span className="text-gray-400">运行:</span>
              <span>{health.uptime_hours.toFixed(0)}h</span>
            </div>
            <div className="flex items-center gap-1.5">
              <AlertTriangle size={12} className="text-tech-orange" />
              <span className="text-gray-400">错误:</span>
              <span>{health.error_count_24h}</span>
            </div>
          </div>
        )}
        <button
          onClick={handleLocate}
          className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-tech-blue/15 text-tech-blue text-sm rounded hover:bg-tech-blue/25 transition-colors"
        >
          <Crosshair size={14} />
          触发定位
        </button>
      </div>
    </div>
  )
}
