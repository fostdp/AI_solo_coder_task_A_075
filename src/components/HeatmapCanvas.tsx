import { useEffect, useRef, useCallback } from 'react'
import { useMap } from 'react-leaflet'
import { useStore } from '@/store'
import { drawHeatmap, drawDetectorMarkers, drawLeakSource, cullDetectorsToViewport, findClickedDetector } from '@/utils/heatmap'

export default function HeatmapCanvas() {
  const map = useMap()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const detectors = useStore((s) => s.detectors)
  const leakSource = useStore((s) => s.leakSource)
  const setSelectedDetector = useStore((s) => s.setSelectedDetector)

  const redraw = useCallback(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const container = map.getContainer()
    const width = container.clientWidth
    const height = container.clientHeight

    canvas.width = width
    canvas.height = height

    const bounds = map.getBounds()
    const mapBounds = {
      north: bounds.getNorth(),
      south: bounds.getSouth(),
      east: bounds.getEast(),
      west: bounds.getWest(),
    }

    const visibleDetectors = cullDetectorsToViewport(detectors, mapBounds)

    const latLngToPixel = (lat: number, lng: number) => {
      const point = map.latLngToContainerPoint([lat, lng])
      return { x: point.x, y: point.y }
    }

    const metersToPixels = (meters: number) => {
      const center = map.getCenter()
      const p1 = map.latLngToContainerPoint([center.lat, center.lng])
      const p2 = map.latLngToContainerPoint([center.lat, center.lng + (meters / 111320) * (1 / Math.cos((center.lat * Math.PI) / 180))])
      return Math.abs(p2.x - p1.x)
    }

    const zoom = map.getZoom()

    ctx.clearRect(0, 0, width, height)
    drawHeatmap(ctx, visibleDetectors, { width, height }, latLngToPixel)
    drawDetectorMarkers(ctx, visibleDetectors, latLngToPixel, zoom)
    if (leakSource) {
      drawLeakSource(ctx, leakSource.source_position, latLngToPixel, metersToPixels)
    }
  }, [map, detectors, leakSource])

  useEffect(() => {
    redraw()
    map.on('moveend zoomend', redraw)
    return () => {
      map.off('moveend zoomend', redraw)
    }
  }, [map, redraw])

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current
      if (!canvas) return
      const rect = canvas.getBoundingClientRect()
      const x = e.clientX - rect.left
      const y = e.clientY - rect.top
      const zoom = map.getZoom()

      const bounds = map.getBounds()
      const mapBounds = {
        north: bounds.getNorth(),
        south: bounds.getSouth(),
        east: bounds.getEast(),
        west: bounds.getWest(),
      }
      const visibleDetectors = cullDetectorsToViewport(detectors, mapBounds)

      const latLngToPixel = (lat: number, lng: number) => {
        const point = map.latLngToContainerPoint([lat, lng])
        return { x: point.x, y: point.y }
      }

      const clicked = findClickedDetector(visibleDetectors, latLngToPixel, x, y, zoom)
      if (clicked) {
        setSelectedDetector(clicked)
      }
    },
    [map, detectors, setSelectedDetector],
  )

  return (
    <canvas
      ref={canvasRef}
      className="absolute inset-0 z-[500] pointer-events-auto"
      style={{ width: '100%', height: '100%' }}
      onClick={handleClick}
    />
  )
}
