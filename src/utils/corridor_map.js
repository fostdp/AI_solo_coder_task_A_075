import { drawHeatmap, drawDetectorMarkers, drawLeakSource, cullDetectorsToViewport, findClickedDetector } from '@/utils/heatmap'

export class CorridorMap {
  constructor(map, canvas) {
    this.map = map
    this.canvas = canvas
    this.detectors = []
    this.leakSource = null
  }

  setDetectors(detectors) {
    this.detectors = detectors
  }

  setLeakSource(source) {
    this.leakSource = source
  }

  getVisibleDetectors() {
    if (!this.map || this.detectors.length === 0) return []
    const bounds = this.map.getBounds()
    const mapBounds = {
      north: bounds.getNorth(),
      south: bounds.getSouth(),
      east: bounds.getEast(),
      west: bounds.getWest(),
    }
    return cullDetectorsToViewport(this.detectors, mapBounds)
  }

  latLngToPixel(lat, lng) {
    const point = this.map.latLngToContainerPoint([lat, lng])
    return { x: point.x, y: point.y }
  }

  metersToPixels(meters) {
    const center = this.map.getCenter()
    const p1 = this.map.latLngToContainerPoint([center.lat, center.lng])
    const p2 = this.map.latLngToContainerPoint([center.lat, center.lng + (meters / 111320) * (1 / Math.cos((center.lat * Math.PI) / 180))])
    return Math.abs(p2.x - p1.x)
  }

  render() {
    const ctx = this.canvas.getContext('2d')
    if (!ctx) return

    const container = this.map.getContainer()
    const width = container.clientWidth
    const height = container.clientHeight

    this.canvas.width = width
    this.canvas.height = height

    const visibleDetectors = this.getVisibleDetectors()
    const zoom = this.map.getZoom()

    ctx.clearRect(0, 0, width, height)
    drawHeatmap(ctx, visibleDetectors, { width, height }, (lat, lng) => this.latLngToPixel(lat, lng))
    drawDetectorMarkers(ctx, visibleDetectors, (lat, lng) => this.latLngToPixel(lat, lng), zoom)
    if (this.leakSource) {
      drawLeakSource(ctx, this.leakSource.source_position, (lat, lng) => this.latLngToPixel(lat, lng), (m) => this.metersToPixels(m))
    }
  }

  findClickedDetector(clickX, clickY) {
    const visibleDetectors = this.getVisibleDetectors()
    const zoom = this.map.getZoom()
    return findClickedDetector(visibleDetectors, (lat, lng) => this.latLngToPixel(lat, lng), clickX, clickY, zoom)
  }

  bindMapEvents(onRedraw) {
    this.map.on('moveend zoomend', onRedraw)
    return () => {
      this.map.off('moveend zoomend', onRedraw)
    }
  }
}
