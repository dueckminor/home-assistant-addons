<template>
  <div ref="mapContainer" class="world-map-container"></div>
</template>

<script>
import 'leaflet/dist/leaflet.css'
import L from 'leaflet'

export default {
  name: 'WorldMap',
  props: {
    locations:   { type: Array,  default: () => [] },
    tileUrl:     { type: String, default: 'https://basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png' },
    attribution: { type: String, default: '© OpenStreetMap contributors © CARTO' }
  },
  data() {
    return { map: null, markers: [], tileLayer: null }
  },
  watch: {
    locations(val) {
      this.updateMarkers(val)
    },
    tileUrl(url) {
      if (this.tileLayer) { this.tileLayer.remove(); this.tileLayer = null }
      this.tileLayer = L.tileLayer(url, { attribution: this.attribution, subdomains: 'abcd', maxZoom: 19, noWrap: true }).addTo(this.map)
    }
  },
  mounted() {
    this.map = L.map(this.$refs.mapContainer, {
      scrollWheelZoom: true,
      worldCopyJump: false,
      maxBounds: [[-90, -180], [90, 180]],
      maxBoundsViscosity: 1.0
    }).setView([20, 0], 2)
    this.tileLayer = L.tileLayer(this.tileUrl, {
      attribution: this.attribution,
      subdomains: 'abcd',
      maxZoom: 19,
      noWrap: true
    }).addTo(this.map)
    this.$nextTick(() => {
      this.map.invalidateSize()
      this.updateMarkers(this.locations)
    })
  },
  beforeUnmount() {
    if (this.map) {
      this.map.remove()
      this.map = null
    }
  },
  methods: {
    invalidateSize() {
      if (this.map) this.map.invalidateSize()
    },
    updateMarkers(locations) {
      if (!this.map) return
      this.markers.forEach(m => m.remove())
      this.markers = []
      if (!locations || !locations.length) return

      const maxCount = Math.max(...locations.map(l => l.count))
      locations.forEach(loc => {
        if (loc.lat == null || loc.lon == null) return
        const radius = Math.max(6, Math.min(30, Math.sqrt(loc.count / maxCount) * 30))
        const marker = L.circleMarker([loc.lat, loc.lon], {
          radius,
          color: '#1976d2',
          fillColor: '#1976d2',
          fillOpacity: 0.6,
          weight: 1
        })
        marker.bindPopup(
          `<strong>${loc.city || '?'}, ${loc.country || '?'}</strong><br>${loc.count.toLocaleString()} requests`
        )
        marker.addTo(this.map)
        this.markers.push(marker)
      })
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
