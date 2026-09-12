<template>
  <div ref="mapContainer" class="world-map-container"></div>
</template>

<script>
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

export default {
  name: 'WorldMap',
  props: {
    locations: { type: Array,  default: () => [] },
    mapStyle:  { type: String, default: 'https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json' }
  },
  data() {
    return { map: null, mapLoaded: false }
  },
  watch: {
    locations(val) {
      this.updateData(val)
    },
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
      zoom: 1.5,
      renderWorldCopies: false
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
  },
  beforeUnmount() {
    if (this.map) { this.map.remove(); this.map = null }
  },
  methods: {
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
            'circle-opacity': 0.6,
            'circle-stroke-width': 1,
            'circle-stroke-color': '#1565c0'
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
      return {
        type: 'FeatureCollection',
        features: locations
          .filter(l => l.lat != null && l.lon != null)
          .map(loc => ({
            type: 'Feature',
            geometry: { type: 'Point', coordinates: [loc.lon, loc.lat] },
            properties: {
              count: loc.count,
              city: loc.city || '?',
              country: loc.country || '?',
              radius: Math.max(6, Math.min(30, Math.sqrt(loc.count / maxCount) * 30))
            }
          }))
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
