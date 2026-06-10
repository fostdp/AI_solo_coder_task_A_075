import { getConcentrationColor } from '@/utils/heatmap'

export class GasPanel {
  constructor(canvas) {
    this.canvas = canvas
  }

  drawTrendChart(points) {
    const ctx = this.canvas.getContext('2d')
    if (!ctx) return

    const W = 280
    const H = 120
    this.canvas.width = W
    this.canvas.height = H

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

  static getConcentrationColor(concentration) {
    return getConcentrationColor(concentration)
  }

  static formatConcentration(value) {
    return value.toFixed(1)
  }

  static getStatusLabel(status) {
    const labels = { online: '在线', fault: '故障', offline: '离线' }
    return labels[status] || status
  }

  static getStatusClass(status) {
    if (status === 'online') return 'bg-tech-green/15 text-tech-green'
    if (status === 'fault') return 'bg-tech-red/15 text-tech-red'
    return 'bg-gray-500/15 text-gray-400'
  }

  static getAlarmLevelLabel(level) {
    return `L${level}`
  }

  static getAlarmLevelBg(level) {
    const bgs = {
      1: 'bg-tech-yellow/15 text-tech-yellow',
      2: 'bg-tech-orange/15 text-tech-orange',
      3: 'bg-tech-red/15 text-tech-red',
    }
    return bgs[level] || ''
  }
}
