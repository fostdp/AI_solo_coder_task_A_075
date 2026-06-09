import { useEffect, useRef, useCallback, useState } from 'react'
import { useStore } from '@/store'
import { api } from '@/utils/api'
import { getConcentrationColor } from '@/utils/heatmap'
import type { DetectorHistory, DetectorHealth } from '@/types'
import { X, Activity, Signal, Wrench, AlertTriangle, Crosshair } from 'lucide-react'

function drawTrendChart(
  canvas: HTMLCanvasElement,
  points: DetectorHistory['points'],
) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const W = 280
  const H = 120
  canvas.width = W
  canvas.height = H

  ctx.fillStyle = '#0a1628'
  ctx.fillRect(0, 0, W, H)

  const padL = 35
  const padR = 10
  const padT = 10
  const padB = 20
  const chartW = W - padL - padR
  const chartH = H - padT - padB

  ctx.strokeStyle = '#1a2d4d'
  ctx.lineWidth = 0.5
  for (let i = 0; i <= 4; i++) {
    const y = padT + (chartH / 4) * i
    ctx.beginPath()
    ctx.moveTo(padL, y)
    ctx.lineTo(padL + chartW, y)
    ctx.stroke()
  }

  ctx.fillStyle = '#4a5568'
  ctx.font = '9px monospace'
  ctx.textAlign = 'right'
  const maxVal = 100
  for (let i = 0; i <= 4; i++) {
    const y = padT + (chartH / 4) * i
    const val = maxVal - (maxVal / 4) * i
    ctx.fillText(`${val}`, padL - 4, y + 3)
  }

  const thresholds = [
    { val: 10, color: '#ffdd00', label: '10%' },
    { val: 20, color: '#ff8800', label: '20%' },
    { val: 50, color: '#ff4444', label: '50%' },
  ]
  for (const t of thresholds) {
    const y = padT + chartH * (1 - t.val / maxVal)
    ctx.strokeStyle = t.color
    ctx.lineWidth = 0.8
    ctx.setLineDash([4, 3])
    ctx.beginPath()
    ctx.moveTo(padL, y)
    ctx.lineTo(padL + chartW, y)
    ctx.stroke()
    ctx.setLineDash([])
    ctx.fillStyle = t.color
    ctx.font = '8px monospace'
    ctx.textAlign = 'left'
    ctx.fillText(t.label, padL + chartW + 2, y + 3)
  }

  if (points.length < 2) return

  ctx.fillStyle = '#4a5568'
  ctx.font = '8px monospace'
  ctx.textAlign = 'center'
  const step = Math.max(1, Math.floor(points.length / 5))
  for (let i = 0; i < points.length; i += step) {
    const x = padL + (chartW / (points.length - 1)) * i
    const t = new Date(points[i].time)
    ctx.fillText(`${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}`, x, H - 4)
  }

  ctx.strokeStyle = '#00d4ff'
  ctx.lineWidth = 1.5
  ctx.beginPath()
  for (let i = 0; i < points.length; i++) {
    const x = padL + (chartW / (points.length - 1)) * i
    const y = padT + chartH * (1 - Math.min(points[i].avg, maxVal) / maxVal)
    if (i === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  }
  ctx.stroke()

  const lastPoint = points[points.length - 1]
  const lastX = padL + chartW
  const lastY = padT + chartH * (1 - Math.min(lastPoint.avg, maxVal) / maxVal)
  ctx.beginPath()
  ctx.arc(lastX, lastY, 3, 0, Math.PI * 2)
  ctx.fillStyle = '#00d4ff'
  ctx.fill()
}

export default function DetectorPopup() {
  const selectedDetector = useStore((s) => s.selectedDetector)
  const setSelectedDetector = useStore((s) => s.setSelectedDetector)
  const setLeakSource = useStore((s) => s.setLeakSource)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [history, setHistory] = useState<DetectorHistory | null>(null)
  const [health, setHealth] = useState<DetectorHealth | null>(null)

  useEffect(() => {
    if (!selectedDetector) return
    const now = new Date()
    const start = new Date(now.getTime() - 60 * 60 * 1000).toISOString()
    api.getDetectorHistory(selectedDetector.id, start, now.toISOString()).then(setHistory).catch(() => {})
    api.getDetectorHealth(selectedDetector.id).then(setHealth).catch(() => {})
  }, [selectedDetector])

  useEffect(() => {
    if (!history || !canvasRef.current) return
    drawTrendChart(canvasRef.current, history.points)
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

  const concColor = getConcentrationColor(selectedDetector.latest_concentration)

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
            {selectedDetector.latest_concentration.toFixed(1)} %LEL
          </span>
        </div>
        <div className="flex justify-between items-center">
          <span className="text-xs text-gray-400">状态</span>
          <span className={`text-xs px-2 py-0.5 rounded ${
            selectedDetector.status === 'online'
              ? 'bg-tech-green/15 text-tech-green'
              : selectedDetector.status === 'fault'
              ? 'bg-tech-red/15 text-tech-red'
              : 'bg-gray-500/15 text-gray-400'
          }`}>
            {selectedDetector.status === 'online' ? '在线' : selectedDetector.status === 'fault' ? '故障' : '离线'}
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
