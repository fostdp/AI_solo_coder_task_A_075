import type { DetectorInfo } from '@/types'

interface HeatmapPoint {
  x: number
  y: number
  intensity: number
}

const CONCENTRATION_COLORS: [number, string][] = [
  [0, '#00ff88'],
  [5, '#44ff44'],
  [10, '#ffdd00'],
  [20, '#ff8800'],
  [50, '#ff4444'],
  [100, '#cc0000'],
]

function getConcentrationColor(concentration: number): string {
  for (let i = CONCENTRATION_COLORS.length - 1; i >= 0; i--) {
    if (concentration >= CONCENTRATION_COLORS[i][0]) {
      return CONCENTRATION_COLORS[i][1]
    }
  }
  return CONCENTRATION_COLORS[0][1]
}

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result
    ? { r: parseInt(result[1], 16), g: parseInt(result[2], 16), b: parseInt(result[3], 16) }
    : { r: 0, g: 255, b: 136 }
}

export function drawHeatmap(
  ctx: CanvasRenderingContext2D,
  detectors: DetectorInfo[],
  mapBounds: { width: number; height: number },
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number }
) {
  ctx.clearRect(0, 0, mapBounds.width, mapBounds.height)

  const points: HeatmapPoint[] = detectors
    .filter(d => d.latest_concentration > 0)
    .map(d => {
      const pixel = latLngToPixel(d.latitude, d.longitude)
      return {
        x: pixel.x,
        y: pixel.y,
        intensity: Math.min(d.latest_concentration / 100, 1),
      }
    })

  const radius = 60

  for (const point of points) {
    const gradient = ctx.createRadialGradient(
      point.x, point.y, 0,
      point.x, point.y, radius
    )
    const color = getConcentrationColor(point.intensity * 100)
    const rgb = hexToRgb(color)
    const alpha = Math.min(point.intensity * 0.8, 0.6)
    gradient.addColorStop(0, `rgba(${rgb.r},${rgb.g},${rgb.b},${alpha})`)
    gradient.addColorStop(0.4, `rgba(${rgb.r},${rgb.g},${rgb.b},${alpha * 0.5})`)
    gradient.addColorStop(1, `rgba(${rgb.r},${rgb.g},${rgb.b},0)`)
    ctx.fillStyle = gradient
    ctx.fillRect(point.x - radius, point.y - radius, radius * 2, radius * 2)
  }
}

export function drawDetectorMarkers(
  ctx: CanvasRenderingContext2D,
  detectors: DetectorInfo[],
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number }
) {
  for (const d of detectors) {
    const pixel = latLngToPixel(d.latitude, d.longitude)
    const color = getConcentrationColor(d.latest_concentration)
    const rgb = hexToRgb(color)

    ctx.beginPath()
    ctx.arc(pixel.x, pixel.y, 4, 0, Math.PI * 2)
    ctx.fillStyle = color
    ctx.fill()

    if (d.latest_concentration >= 10) {
      ctx.beginPath()
      ctx.arc(pixel.x, pixel.y, 8, 0, Math.PI * 2)
      ctx.strokeStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.4)`
      ctx.lineWidth = 1.5
      ctx.stroke()
    }

    if (d.latest_concentration >= 20) {
      ctx.beginPath()
      ctx.arc(pixel.x, pixel.y, 14, 0, Math.PI * 2)
      ctx.strokeStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.2)`
      ctx.lineWidth = 1
      ctx.stroke()
    }
  }
}

export function drawLeakSource(
  ctx: CanvasRenderingContext2D,
  source: { lat: number; lng: number; diffusion_radius: number },
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number },
  metersToPixels: (meters: number) => number
) {
  const pixel = latLngToPixel(source.lat, source.lng)
  const radiusPixels = metersToPixels(source.diffusion_radius)

  const gradient = ctx.createRadialGradient(pixel.x, pixel.y, 0, pixel.x, pixel.y, radiusPixels)
  gradient.addColorStop(0, 'rgba(255, 68, 68, 0.4)')
  gradient.addColorStop(0.6, 'rgba(255, 68, 68, 0.15)')
  gradient.addColorStop(1, 'rgba(255, 68, 68, 0)')

  ctx.fillStyle = gradient
  ctx.beginPath()
  ctx.arc(pixel.x, pixel.y, radiusPixels, 0, Math.PI * 2)
  ctx.fill()

  ctx.strokeStyle = 'rgba(255, 68, 68, 0.8)'
  ctx.lineWidth = 2
  ctx.setLineDash([6, 4])
  ctx.beginPath()
  ctx.arc(pixel.x, pixel.y, radiusPixels, 0, Math.PI * 2)
  ctx.stroke()
  ctx.setLineDash([])

  ctx.beginPath()
  ctx.arc(pixel.x, pixel.y, 6, 0, Math.PI * 2)
  ctx.fillStyle = '#ff4444'
  ctx.fill()
  ctx.strokeStyle = '#ffffff'
  ctx.lineWidth = 1.5
  ctx.stroke()
}

export { getConcentrationColor, hexToRgb }
