<template>
  <div ref="mapContainer" class="world-map-container"></div>
</template>

<script>
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

export default {
  name: 'WorldMap',
  props: {
    locations:    { type: Array,  default: () => [] },
    mapStyle:     { type: String, default: 'https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json' },
    highlightLat: { type: Number, default: null },
    highlightLon: { type: Number, default: null }
  },
  data() {
    return { map: null, mapLoaded: false }
  },
  watch: {
    locations(val) {
      this.updateData(val)
    },
    highlightLat() { this.updateData(this.locations) },
    highlightLon()  { this.updateData(this.locations) },
    mapStyle(style) {
      if (!this.map) return
      this.mapLoaded = false
      this.map.setStyle(style)
      this.map.once('style.load', () => {
        this.mapLoaded = true
        this.addDataLayer()
        this.updateData(this.locations)
      })
    }
  },
  mounted() {
    this.map = new maplibregl.Map({
      container: this.$refs.mapContainer,
      style: this.mapStyle,
      center: [0, 20],
      zoom: 1,
      minZoom: -2,
    })
    this.map.on('load', () => {
      this.mapLoaded = true
      this.addDataLayer()
      this.updateData(this.locations)
    })
    this.map.on('click', 'locations', e => {
      const p = e.features[0].properties
      new maplibregl.Popup()
        .setLngLat(e.lngLat)
        .setHTML(`<strong>${p.city}, ${p.country}</strong><br>${Number(p.count).toLocaleString()} requests`)
        .addTo(this.map)
    })
    this.map.on('mouseenter', 'locations', () => { this.map.getCanvas().style.cursor = 'pointer' })
    this.map.on('mouseleave', 'locations', () => { this.map.getCanvas().style.cursor = '' })

    this._initialFitDone = false
    this._resizeObserver = new ResizeObserver(() => {
      if (!this.map) return
      this.map.resize()
      if (!this._initialFitDone) {
        const el = this.$refs.mapContainer
        if (el.clientWidth > 0 && el.clientHeight > 0) {
          this._initialFitDone = true
          this.fitWorld()
        }
      }
    })
    this._resizeObserver.observe(this.$refs.mapContainer)
  },
  beforeUnmount() {
    if (this._resizeObserver) { this._resizeObserver.disconnect(); this._resizeObserver = null }
    if (this.map) { this.map.remove(); this.map = null }
  },
  methods: {
    fitWorld() {
      const el = this.$refs.mapContainer
      if (!el || !el.clientWidth) return
      // Zoom so the world fills the container width — prevents world copies in the initial view.
      // (fitBounds with world bounds would be height-constrained on wide containers, showing copies.)
      const zoom = Math.log2(el.clientWidth / 512)
      this.map.jumpTo({ center: [0, 20], zoom })
    },
    invalidateSize() {
      if (this.map) this.map.resize()
    },
    addDataLayer() {
      if (!this.map || !this.mapLoaded) return
      if (!this.map.getSource('locations')) {
        this.map.addSource('locations', {
          type: 'geojson',
          data: { type: 'FeatureCollection', features: [] }
        })
      }
      if (!this.map.getLayer('locations')) {
        this.map.addLayer({
          id: 'locations',
          type: 'circle',
          source: 'locations',
          paint: {
            'circle-radius': ['get', 'radius'],
            'circle-color': '#1976d2',
            'circle-opacity': ['get', 'opacity'],
            'circle-stroke-width': 1,
            'circle-stroke-color': '#1565c0',
            'circle-stroke-opacity': ['get', 'opacity']
          }
        })
      }
    },
    updateData(locations) {
      if (!this.map || !this.mapLoaded) return
      const source = this.map.getSource('locations')
      if (source) source.setData(this.toGeoJSON(locations))
    },
    toGeoJSON(locations) {
      if (!locations || !locations.length) return { type: 'FeatureCollection', features: [] }
      const maxCount = Math.max(...locations.map(l => l.count))
      const hasHighlight = this.highlightLat !== null && this.highlightLon !== null
      return {
        type: 'FeatureCollection',
        features: locations
          .filter(l => l.lat != null && l.lon != null)
          .map(loc => {
            const isHighlighted = hasHighlight && loc.lat === this.highlightLat && loc.lon === this.highlightLon
            const opacity = hasHighlight ? (isHighlighted ? 0.8 : 0.2) : 0.7
            return {
              type: 'Feature',
              geometry: { type: 'Point', coordinates: [loc.lon, loc.lat] },
              properties: {
                count: loc.count,
                city: loc.city || '?',
                country: loc.country || '?',
                radius: Math.max(6, Math.min(30, Math.sqrt(loc.count / maxCount) * 30)),
                opacity
              }
            }
          })
      }
    }
  }
}
</script>

<style scoped>
.world-map-container {
  width: 100%;
  height: 100%;
  min-height: 300px;
}
</style>
