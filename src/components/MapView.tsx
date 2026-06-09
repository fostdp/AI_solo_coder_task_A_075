import { MapContainer, TileLayer } from 'react-leaflet'
import TunnelPolyline from './TunnelPolyline'
import HeatmapCanvas from './HeatmapCanvas'
import DetectorPopup from './DetectorPopup'
import LeakSourceOverlay from './LeakSourceOverlay'
import { useStore } from '@/store'
import 'leaflet/dist/leaflet.css'

const MAP_CENTER: [number, number] = [31.2320, 121.5800]
const MAP_ZOOM = 13

export default function MapView() {
  const leakSource = useStore((s) => s.leakSource)

  return (
    <div className="relative w-full h-full">
      <MapContainer
        center={MAP_CENTER}
        zoom={MAP_ZOOM}
        className="w-full h-full"
        zoomControl={true}
        attributionControl={false}
      >
        <TileLayer
          url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
          maxZoom={19}
        />
        <TunnelPolyline />
        <HeatmapCanvas />
        {leakSource && <LeakSourceOverlay />}
      </MapContainer>
      <DetectorPopup />
    </div>
  )
}
