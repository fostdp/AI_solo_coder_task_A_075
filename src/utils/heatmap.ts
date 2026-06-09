import type { DetectorInfo } from '@/types'

interface HeatmapPoint {
  x: number
  y: number
  intensity: number
}

interface Cluster {
  x: number
  y: number
  count: number
  maxConcentration: number
  detectors: DetectorInfo[]
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

export function cullDetectorsToViewport(
  detectors: DetectorInfo[],
  bounds: { north: number; south: number; east: number; west: number },
  padding: number = 0.002
): DetectorInfo[] {
  return detectors.filter(d =>
    d.latitude >= bounds.south - padding &&
    d.latitude <= bounds.north + padding &&
    d.longitude >= bounds.west - padding &&
    d.longitude <= bounds.east + padding
  )
}

export function clusterDetectors(
  detectors: DetectorInfo[],
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number },
  zoom: number
): { individuals: DetectorInfo[]; clusters: Cluster[] } {
  const clusterRadius = getClusterRadius(zoom)

  if (clusterRadius <= 0) {
    return { individuals: detectors, clusters: [] }
  }

  const visited = new Set<number>()
  const clusters: Cluster[] = []
  const individuals: DetectorInfo[] = []

  for (let i = 0; i < detectors.length; i++) {
    if (visited.has(i)) continue
    visited.add(i)

    const d = detectors[i]
    const pixel = latLngToPixel(d.latitude, d.longitude)
    const group: DetectorInfo[] = [d]

    for (let j = i + 1; j < detectors.length; j++) {
      if (visited.has(j)) continue
      const other = detectors[j]
      const otherPixel = latLngToPixel(other.latitude, other.longitude)
      const dx = pixel.x - otherPixel.x
      const dy = pixel.y - otherPixel.y
      if (dx * dx + dy * dy < clusterRadius * clusterRadius) {
        visited.add(j)
        group.push(other)
      }
    }

    if (group.length === 1) {
      individuals.push(group[0])
    } else {
      let sumX = 0, sumY = 0, maxConc = 0
      for (const det of group) {
        const p = latLngToPixel(det.latitude, det.longitude)
        sumX += p.x
        sumY += p.y
        if (det.latest_concentration > maxConc) maxConc = det.latest_concentration
      }
      clusters.push({
        x: sumX / group.length,
        y: sumY / group.length,
        count: group.length,
        maxConcentration: maxConc,
        detectors: group,
      })
    }
  }

  return { individuals, clusters }
}

function getClusterRadius(zoom: number): number {
  if (zoom >= 15) return 0
  if (zoom >= 13) return 10
  if (zoom >= 11) return 18
  if (zoom >= 9) return 28
  return 40
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
    if (point.x < -radius || point.x > mapBounds.width + radius ||
        point.y < -radius || point.y > mapBounds.height + radius) {
      continue
    }

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
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number },
  zoom: number
) {
  const { individuals, clusters } = clusterDetectors(detectors, latLngToPixel, zoom)

  for (const d of individuals) {
    const pixel = latLngToPixel(d.latitude, d.longitude)
    if (pixel.x < -20 || pixel.x > ctx.canvas.width + 20 ||
        pixel.y < -20 || pixel.y > ctx.canvas.height + 20) {
      continue
    }

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

  for (const cluster of clusters) {
    if (cluster.x < -20 || cluster.x > ctx.canvas.width + 20 ||
        cluster.y < -20 || cluster.y > ctx.canvas.height + 20) {
      continue
    }

    const color = getConcentrationColor(cluster.maxConcentration)
    const rgb = hexToRgb(color)
    const radius = Math.min(6 + Math.sqrt(cluster.count) * 2, 20)

    ctx.beginPath()
    ctx.arc(cluster.x, cluster.y, radius, 0, Math.PI * 2)
    ctx.fillStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.6)`
    ctx.fill()
    ctx.strokeStyle = `rgba(${rgb.r},${rgb.g},${rgb.b},0.8)`
    ctx.lineWidth = 1.5
    ctx.stroke()

    ctx.fillStyle = '#ffffff'
    ctx.font = '10px Inter, sans-serif'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(String(cluster.count), cluster.x, cluster.y)
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

export function findClickedDetector(
  detectors: DetectorInfo[],
  latLngToPixel: (lat: number, lng: number) => { x: number; y: number },
  clickX: number,
  clickY: number,
  zoom: number
): DetectorInfo | null {
  const { individuals, clusters } = clusterDetectors(detectors, latLngToPixel, zoom)

  for (const d of individuals) {
    const point = latLngToPixel(d.latitude, d.longitude)
    const dx = point.x - clickX
    const dy = point.y - clickY
    if (dx * dx + dy * dy <= 100) {
      return d
    }
  }

  for (const cluster of clusters) {
    const dx = cluster.x - clickX
    const dy = cluster.y - clickY
    const radius = Math.min(6 + Math.sqrt(cluster.count) * 2, 20)
    if (dx * dx + dy * dy <= radius * radius) {
      return cluster.detectors.reduce((prev, curr) =>
        prev.latest_concentration > curr.latest_concentration ? prev : curr
      )
    }
  }

  return null
}

export { getConcentrationColor, hexToRgb }
