<template>
  <div>
    <!-- Filter bar -->
    <v-row class="mb-4 align-center" dense>
      <v-col cols="12" sm="4">
        <v-select
          v-model="selectedHostname"
          :items="hostnameItems"
          label="Hostname"
          clearable
          density="compact"
          hide-details
          @update:model-value="loadData"
        />
      </v-col>
      <v-col cols="12" sm="3">
        <v-text-field
          v-model="fromDate"
          label="From"
          type="date"
          density="compact"
          hide-details
          @change="loadData"
        />
      </v-col>
      <v-col cols="12" sm="3">
        <v-text-field
          v-model="toDate"
          label="To"
          type="date"
          density="compact"
          hide-details
          @change="loadData"
        />
      </v-col>
      <v-col cols="12" sm="2">
        <v-btn-toggle v-model="granularity" density="compact" mandatory @update:model-value="loadData">
          <v-btn value="hour" size="small">Hour</v-btn>
          <v-btn value="day" size="small">Day</v-btn>
        </v-btn-toggle>
      </v-col>
    </v-row>

    <!-- World map + IP list + chart -->
    <v-row class="mb-4" dense>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Access Locations</v-card-title>
          <v-card-text class="pa-0">
            <WorldMap ref="worldMap" :locations="mapData" :map-style="mapStyle" :highlight-lat="highlightLat" :highlight-lon="highlightLon" style="height: 400px" />
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Top IPs</v-card-title>
          <v-data-table
            :headers="ipHeaders"
            :items="ipData"
            :items-per-page="10"
            density="compact"
            style="max-height: 430px; overflow-y: auto"
            :row-props="({ item }) => ({ class: item.ip === selectedIP ? 'bg-primary-lighten-5' : 'cursor-pointer', style: 'cursor: pointer' })"
            @click:row="(_, { item }) => toggleIP(item.ip)"
          >
            <template #item.ip="{ item }">
              <span class="text-caption" :class="item.ip === selectedIP ? 'font-weight-bold' : ''">{{ item.ip }}</span>
            </template>
            <template #item.location="{ item }">
              <span class="text-caption">
                {{ countryFlag(item.country_code) }}
                {{ [item.city, item.country].filter(Boolean).join(', ') }}
              </span>
            </template>
          </v-data-table>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Request Volume</v-card-title>
          <v-card-text style="height: 400px; padding-bottom: 8px">
            <AccessChart :data-points="chartData" :granularity="granularity" style="height: 100%" />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Top paths table -->
    <v-card>
      <v-card-title class="text-subtitle-1">Top Paths</v-card-title>
      <v-data-table
        :headers="pathHeaders"
        :items="pathData"
        :items-per-page="20"
        density="compact"
      >
        <template #item.errors="{ item }">
          <span :class="item.errors > 0 ? 'text-error' : ''">{{ item.errors }}</span>
        </template>
      </v-data-table>
    </v-card>
  </div>
</template>

<script>
import WorldMap from '../WorldMap.vue'
import AccessChart from '../AccessChart.vue'
import { apiGet } from '../../../../shared/utils/homeassistant.js'

export default {
  name: 'MetricsTab',
  components: { WorldMap, AccessChart },
  props: {
    isActive: { type: Boolean, default: false }
  },
  data() {
    const now = new Date()
    const weekAgo = new Date(now)
    weekAgo.setDate(weekAgo.getDate() - 7)
    return {
      hostnames: [],
      selectedHostname: '',
      fromDate: weekAgo.toISOString().slice(0, 10),
      toDate: now.toISOString().slice(0, 10),
      granularity: 'hour',
      selectedIP: '',
      mapData: [],
      chartData: [],
      pathData: [],
      ipData: [],
      ipHeaders: [
        { title: 'IP',       key: 'ip',       sortable: true },
        { title: 'Location', key: 'location', sortable: false },
        { title: 'Requests', key: 'count',    sortable: true }
      ],
      pathHeaders: [
        { title: 'Method',   key: 'method',   sortable: true },
        { title: 'Hostname', key: 'hostname', sortable: true },
        { title: 'Path',     key: 'path',     sortable: true },
        { title: 'Requests', key: 'count',    sortable: true },
        { title: 'Errors',   key: 'errors',   sortable: true }
      ],
      mapStyle: 'https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json'
    }
  },
  computed: {
    hostnameItems() {
      return [{ title: 'All hostnames', value: '' }, ...this.hostnames.map(h => ({ title: h, value: h }))]
    },
    highlightLat() {
      if (!this.selectedIP) return null
      const entry = this.ipData.find(d => d.ip === this.selectedIP)
      return entry ? entry.lat : null
    },
    highlightLon() {
      if (!this.selectedIP) return null
      const entry = this.ipData.find(d => d.ip === this.selectedIP)
      return entry ? entry.lon : null
    }
  },
  watch: {
  },
  async mounted() {
    await this.loadTileConfig()
    await this.loadHostnames()
    await this.loadData()
  },
  methods: {
    countryFlag(code) {
      if (!code || code.length !== 2) return ''
      return String.fromCodePoint(
        ...code.toUpperCase().split('').map(c => 0x1F1E6 + c.charCodeAt(0) - 65)
      )
    },
    toggleIP(ip) {
      this.selectedIP = this.selectedIP === ip ? '' : ip
      this.loadData()
    },
    clearIP() {
      this.selectedIP = ''
      this.loadData()
    },
    async loadTileConfig() {
      try {
        const data = await apiGet('metrics/config')
        if (data && data.style_url) this.mapStyle = data.style_url
      } catch { /* use default */ }
    },
    async loadHostnames() {
      try {
        const data = await apiGet('metrics/hostnames')
        this.hostnames = Array.isArray(data) ? data : []
      } catch {
        this.hostnames = []
      }
    },
    async loadData() {
      const from = new Date(this.fromDate).toISOString()
      const to = new Date(this.toDate + 'T23:59:59').toISOString()
      const hn = this.selectedHostname ? `&hostname=${encodeURIComponent(this.selectedHostname)}` : ''
      const ip = this.selectedIP ? `&ip=${encodeURIComponent(this.selectedIP)}` : ''

      const [map, ts, paths, ips] = await Promise.all([
        apiGet(`metrics/map?from=${from}&to=${to}${hn}`).catch(() => []),
        apiGet(`metrics/timeseries?from=${from}&to=${to}&granularity=${this.granularity}${hn}${ip}`).catch(() => []),
        apiGet(`metrics/paths?from=${from}&to=${to}${hn}${ip}`).catch(() => []),
        apiGet(`metrics/ips?from=${from}&to=${to}${hn}`).catch(() => [])
      ])
      this.mapData = Array.isArray(map) ? map : []
      this.chartData = Array.isArray(ts) ? ts : []
      this.pathData = Array.isArray(paths) ? paths : []
      this.ipData = Array.isArray(ips) ? ips : []
    }
  }
}
</script>
