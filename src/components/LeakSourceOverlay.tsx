import { CircleMarker, Tooltip } from 'react-leaflet'
import { useStore } from '@/store'

export default function LeakSourceOverlay() {
  const leakSource = useStore((s) => s.leakSource)
  if (!leakSource) return null

  const { lat, lng } = leakSource.source_position

  return (
    <>
      <CircleMarker
        center={[lat, lng]}
        radius={8}
        pathOptions={{
          color: '#ff4444',
          fillColor: '#ff4444',
          fillOpacity: 0.8,
          weight: 2,
        }}
      >
        <Tooltip permanent direction="top" offset={[0, -10]}>
          <div className="text-xs">
            <div className="font-bold text-red-400">泄漏源</div>
            <div>置信度: {(leakSource.confidence * 100).toFixed(1)}%</div>
            <div>泄漏率: {leakSource.leak_rate.toFixed(2)} L/min</div>
            <div>扩散半径: {leakSource.diffusion_radius.toFixed(0)} m</div>
          </div>
        </Tooltip>
      </CircleMarker>
      <CircleMarker
        center={[lat, lng]}
        radius={16}
        pathOptions={{
          color: '#ff4444',
          fillColor: 'transparent',
          weight: 1.5,
          dashArray: '4 4',
          className: 'leak-pulse',
        }}
      />
    </>
  )
}
