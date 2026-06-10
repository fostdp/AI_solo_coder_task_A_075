import { useEffect, useRef, useCallback } from 'react'
import { useMap } from 'react-leaflet'
import { useStore } from '@/store'
import { CorridorMap } from '@/utils/corridor_map'

export default function HeatmapCanvas() {
  const map = useMap()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const corridorMapRef = useRef<CorridorMap | null>(null)
  const detectors = useStore((s) => s.detectors)
  const leakSource = useStore((s) => s.leakSource)
  const setSelectedDetector = useStore((s) => s.setSelectedDetector)

  useEffect(() => {
    if (canvasRef.current && map) {
      corridorMapRef.current = new CorridorMap(map, canvasRef.current)
    }
  }, [map])

  const redraw = useCallback(() => {
    if (!corridorMapRef.current) return
    corridorMapRef.current.setDetectors(detectors)
    corridorMapRef.current.setLeakSource(leakSource)
    corridorMapRef.current.render()
  }, [detectors, leakSource])

  useEffect(() => {
    redraw()
    const cleanup = corridorMapRef.current?.bindMapEvents(redraw)
    return () => {
      cleanup?.()
    }
  }, [map, redraw])

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      if (!corridorMapRef.current) return
      const canvas = canvasRef.current
      if (!canvas) return
      const rect = canvas.getBoundingClientRect()
      const x = e.clientX - rect.left
      const y = e.clientY - rect.top
      const clicked = corridorMapRef.current.findClickedDetector(x, y)
      if (clicked) {
        setSelectedDetector(clicked)
      }
    },
    [setSelectedDetector],
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
