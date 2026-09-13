<template>
  <div ref="mapContainer" class="world-map-container"></div>
</template>

<script>
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

function sectorPath(cx, cy, r, a1, a2) {
  const x1 = cx + r * Math.cos(a1), y1 = cy + r * Math.sin(a1)
  const x2 = cx + r * Math.cos(a2), y2 = cy + r * Math.sin(a2)
  return `M${cx},${cy} L${x1.toFixed(2)},${y1.toFixed(2)} A${r},${r} 0 ${a2 - a1 > Math.PI ? 1 : 0},1 ${x2.toFixed(2)},${y2.toFixed(2)} Z`
}

function makePieSVG(segments, r, opacity) {
  const total = segments.reduce((s, x) => s + x.value, 0)
  if (total === 0) return ''
  const active = segments.filter(s => s.value > 0)
  const d = r * 2
  let paths
  if (active.length === 1) {
    paths = [`<circle cx="${r}" cy="${r}" r="${r}" fill="${active[0].color}"/>`]
  } else {
    let angle = -Math.PI / 2
    paths = active.map(seg => {
      const sweep = (seg.value / total) * 2 * Math.PI
      const path = `<path d="${sectorPath(r, r, r, angle, angle + sweep)}" fill="${seg.color}"/>`
      angle += sweep
      return path
    })
  }
  return `<svg width="${d}" height="${d}" style="opacity:${opacity};display:block;overflow:visible">${paths.join('')}<circle cx="${r}" cy="${r}" r="${r}" fill="none" stroke="white" stroke-width="1.5"/></svg>`
}

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
    locations()    { this.updateData(this.locations) },
    highlightLat() { this.updateData(this.locations) },
    highlightLon() { this.updateData(this.locations) },
    mapStyle(style) {
      if (!this.map) return
      this.mapLoaded = false
      this.map.setStyle(style)
      this.map.once('style.load', () => {
        this.mapLoaded = true
        this.updateData(this.locations)
      })
    }
  },
  mounted() {
    this._markers = []
    this.map = new maplibregl.Map({
      container: this.$refs.mapContainer,
      style: this.mapStyle,
      center: [0, 20],
      zoom: 1,
      minZoom: -2,
    })
    this.map.on('load', () => {
      this.mapLoaded = true
      this.updateData(this.locations)
    })

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
    this._markers.forEach(m => m.remove())
    this._markers = []
    if (this.map) { this.map.remove(); this.map = null }
  },
  methods: {
    fitWorld() {
      const el = this.$refs.mapContainer
      if (!el || !el.clientWidth) return
      const zoom = Math.log2(el.clientWidth / 512)
      this.map.jumpTo({ center: [0, 20], zoom })
    },
    invalidateSize() {
      if (this.map) this.map.resize()
    },
    updateData(locations) {
      if (!this.map || !this.mapLoaded) return

      this._markers.forEach(m => m.remove())
      this._markers = []

      if (!locations || !locations.length) return

      const totals = locations.map(l => (l.success || 0) + (l.errors || 0) + (l.blocked || 0))
      const maxCount = Math.max(...totals)
      const hasHighlight = this.highlightLat !== null && this.highlightLon !== null

      for (let i = 0; i < locations.length; i++) {
        const loc = locations[i]
        if (loc.lat == null || loc.lon == null) continue

        const total = totals[i]
        if (total === 0) continue

        const isHighlighted = !hasHighlight || (loc.lat === this.highlightLat && loc.lon === this.highlightLon)
        const opacity = isHighlighted ? 1 : 0.2
        const radius = Math.max(8, Math.min(30, Math.sqrt(total / maxCount) * 30))

        const segments = [
          { value: loc.success || 0, color: '#43a047' },
          { value: loc.errors  || 0, color: '#e53935' },
          { value: loc.blocked || 0, color: '#fb8c00' }
        ]

        const el = document.createElement('div')
        el.innerHTML = makePieSVG(segments, radius, opacity)
        el.style.cursor = 'pointer'

        const popup = new maplibregl.Popup({ offset: radius + 4 }).setHTML(
          `<strong>${loc.city || '?'}, ${loc.country || '?'}</strong><br>` +
          `<span style="color:#43a047">&#9679;</span> Success: ${(loc.success || 0).toLocaleString()}<br>` +
          `<span style="color:#e53935">&#9679;</span> Errors: ${(loc.errors || 0).toLocaleString()}<br>` +
          `<span style="color:#fb8c00">&#9679;</span> Blocked: ${(loc.blocked || 0).toLocaleString()}`
        )

        const marker = new maplibregl.Marker({ element: el, anchor: 'center' })
          .setLngLat([loc.lon, loc.lat])
          .setPopup(popup)
          .addTo(this.map)

        this._markers.push(marker)
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
