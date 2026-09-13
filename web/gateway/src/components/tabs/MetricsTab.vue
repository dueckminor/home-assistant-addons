<template>
  <div>
    <!-- Filter bar -->
    <v-row class="mb-2 align-center" dense>
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

    <!-- Category checkboxes -->
    <v-row class="mb-4 align-center" dense>
      <v-col cols="auto">
        <v-checkbox v-model="showSuccess" density="compact" hide-details color="#43a047">
          <template #label><span style="color:#43a047">Success</span></template>
        </v-checkbox>
      </v-col>
      <v-col cols="auto">
        <v-checkbox v-model="showErrors" density="compact" hide-details color="#e53935">
          <template #label><span style="color:#e53935">Errors</span></template>
        </v-checkbox>
      </v-col>
      <v-col cols="auto">
        <v-checkbox v-model="showBlocked" density="compact" hide-details color="#fb8c00">
          <template #label><span style="color:#fb8c00">Blocked</span></template>
        </v-checkbox>
      </v-col>
    </v-row>

    <!-- World map + IP list + chart -->
    <v-row class="mb-4" dense>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Access Locations</v-card-title>
          <v-card-text class="pa-0">
            <WorldMap
              ref="worldMap"
              :locations="mapData"
              :map-style="mapStyle"
              :highlight-lat="highlightLat"
              :highlight-lon="highlightLon"
              :show-success="showSuccess"
              :show-errors="showErrors"
              :show-blocked="showBlocked"
              style="height: 400px"
              @click-location="toggleLocation($event.lat, $event.lon, $event.city, $event.country)"
            />
          </v-card-text>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Top IPs</v-card-title>

          <!-- Card layout: shown below custom breakpoint -->
          <div class="ip-compact">
            <v-divider />
            <div style="max-height: 430px; overflow-y: auto">
              <div
                v-for="(item, idx) in filteredIPData"
                :key="item.ip"
                class="d-flex justify-space-between align-start px-3 py-2"
                :class="(item.ip === selectedIP || isLocationSelected(item)) ? 'bg-primary-lighten-5' : ''"
                :style="{ borderTop: idx > 0 ? '1px solid rgba(0,0,0,0.08)' : 'none', cursor: 'pointer' }"
                @click="toggleIP(item.ip)"
              >
                <div class="d-flex flex-column" style="min-width: 0; overflow: hidden">
                  <span class="text-body-2" :class="item.ip === selectedIP ? 'font-weight-bold' : ''">{{ item.ip }}</span>
                  <span
                    class="text-caption text-medium-emphasis"
                    :class="isLocationSelected(item) ? 'font-weight-bold' : ''"
                    @click.stop="toggleLocation(item.lat, item.lon, item.city, item.country)"
                  >{{ item.city }}</span>
                  <span
                    class="text-caption text-medium-emphasis"
                    :class="isLocationSelected(item) ? 'font-weight-bold' : ''"
                    @click.stop="toggleLocation(item.lat, item.lon, item.city, item.country)"
                  >{{ countryFlag(item.country_code) }} {{ item.country }}</span>
                </div>
                <div class="d-flex flex-column align-end text-caption ml-2" style="white-space: nowrap; flex-shrink: 0">
                  <span :style="showSuccess ? 'color:#43a047' : 'visibility:hidden'">{{ (item.success || 0).toLocaleString() }}</span>
                  <span :style="showErrors  ? 'color:#e53935' : 'visibility:hidden'">{{ (item.errors  || 0).toLocaleString() }}</span>
                  <span :style="showBlocked ? 'color:#fb8c00' : 'visibility:hidden'">{{ (item.blocked || 0).toLocaleString() }}</span>
                </div>
              </div>
              <div v-if="!filteredIPData.length" class="text-caption text-medium-emphasis text-center pa-4">No data</div>
            </div>
          </div>

          <!-- Table layout: shown above custom breakpoint -->
          <v-data-table
            class="ip-table compact-table"
            :headers="ipHeaders"
            :items="filteredIPData"
            :items-per-page="-1"
            hide-default-footer
            density="compact"
            style="max-height: 430px; overflow-y: auto"
            :row-props="rowProps"
            @click:row="(_, { item }) => toggleIP(item.ip)"
          >
            <template #item.ip="{ item }">
              <span class="text-caption" :class="item.ip === selectedIP ? 'font-weight-bold' : ''">{{ item.ip }}</span>
            </template>
            <template #item.location="{ item }">
              <span
                class="text-caption"
                :class="isLocationSelected(item) ? 'font-weight-bold' : ''"
                style="cursor: pointer"
                @click.stop="toggleLocation(item.lat, item.lon, item.city, item.country)"
              >
                {{ countryFlag(item.country_code) }}
                {{ [item.city, item.country].filter(Boolean).join(', ') }}
              </span>
            </template>
            <template #item.success="{ item }">
              <span style="color:#43a047">{{ (item.success || 0).toLocaleString() }}</span>
            </template>
            <template #item.errors="{ item }">
              <span style="color:#e53935">{{ (item.errors || 0).toLocaleString() }}</span>
            </template>
            <template #item.blocked="{ item }">
              <span style="color:#fb8c00">{{ (item.blocked || 0).toLocaleString() }}</span>
            </template>
          </v-data-table>
        </v-card>
      </v-col>
      <v-col cols="12" md="4">
        <v-card height="100%">
          <v-card-title class="text-subtitle-1">Request Volume</v-card-title>
          <v-card-text style="height: 400px; padding-bottom: 8px">
            <AccessChart :data-points="chartData" :granularity="granularity" :show-success="showSuccess" :show-errors="showErrors" :show-blocked="showBlocked" style="height: 100%" />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Top paths table -->
    <v-card>
      <v-card-title class="text-subtitle-1">Top Paths</v-card-title>

      <!-- Card layout for small screens -->
      <div class="path-compact">
        <v-divider />
        <div style="max-height: 430px; overflow-y: auto">
          <div
            v-for="(item, idx) in filteredPathData"
            :key="`${item.hostname}-${item.method}-${item.path}`"
            class="d-flex justify-space-between align-start px-3 py-2"
            :style="{ borderTop: idx > 0 ? '1px solid rgba(0,0,0,0.08)' : 'none' }"
          >
            <div class="d-flex flex-column text-caption" style="min-width: 0; overflow: hidden">
              <span :style="item.blocked ? 'color:#fb8c00' : ''">{{ item.blocked ? '—' : item.method }}</span>
              <span :style="item.blocked ? 'color:#fb8c00' : 'color: rgba(0,0,0,0.6)'">{{ item.hostname }}</span>
              <span
                class="text-medium-emphasis"
                :style="item.blocked ? 'color:#fb8c00' : ''"
                style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap"
              >{{ item.blocked ? '—' : item.path }}</span>
            </div>
            <div class="d-flex flex-column align-end text-caption ml-2" style="white-space: nowrap; flex-shrink: 0">
              <span :style="showSuccess ? 'color:#43a047' : 'visibility:hidden'">{{ item.blocked ? 0 : (item.count - item.errors).toLocaleString() }}</span>
              <span :style="showErrors  ? 'color:#e53935' : 'visibility:hidden'">{{ (item.errors  || 0).toLocaleString() }}</span>
              <span :style="showBlocked ? 'color:#fb8c00' : 'visibility:hidden'">{{ item.blocked ? item.count.toLocaleString() : 0 }}</span>
            </div>
          </div>
          <div v-if="!filteredPathData.length" class="text-caption text-medium-emphasis text-center pa-4">No data</div>
        </div>
      </div>

      <!-- Table layout for medium+ screens -->
      <v-data-table
        class="path-table compact-table"
        :headers="pathHeaders"
        :items="filteredPathData"
        :items-per-page="-1"
        hide-default-footer
        density="compact"
        style="max-height: 430px; overflow-y: auto"
      >
        <template #item.method="{ item }">
          <span v-if="item.blocked" style="color:#fb8c00">—</span>
          <span v-else>{{ item.method }}</span>
        </template>
        <template #item.path="{ item }">
          <span v-if="item.blocked" style="color:#fb8c00">—</span>
          <span v-else>{{ item.path }}</span>
        </template>
        <template #item.hostname="{ item }">
          <span :style="item.blocked ? 'color:#fb8c00' : ''">{{ item.hostname }}</span>
        </template>
        <template #item.pathSuccess="{ item }">
          <span style="color:#43a047">{{ item.blocked ? 0 : (item.count - item.errors).toLocaleString() }}</span>
        </template>
        <template #item.errors="{ item }">
          <span style="color:#e53935">{{ (item.errors || 0).toLocaleString() }}</span>
        </template>
        <template #item.pathBlocked="{ item }">
          <span style="color:#fb8c00">{{ item.blocked ? item.count.toLocaleString() : 0 }}</span>
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
      showSuccess: true,
      showErrors: true,
      showBlocked: true,
      selectedIP: '',
      selectedLocation: null,  // { lat, lon, city, country }
      mapData: [],
      chartData: [],
      pathData: [],
      ipData: [],
      mapStyle: 'https://basemaps.cartocdn.com/gl/voyager-gl-style/style.json'
    }
  },
  computed: {
    hostnameItems() {
      return [{ title: 'All hostnames', value: '' }, ...this.hostnames.map(h => ({ title: h, value: h }))]
    },
    ipHeaders() {
      const headers = [
        { title: 'IP',       key: 'ip',       sortable: true,  width: '120px' },
        { title: 'Location', key: 'location', sortable: false }
      ]
      if (this.showSuccess) headers.push({ title: 'Success', key: 'success', sortable: true, width: '40px' })
      if (this.showErrors)  headers.push({ title: 'Errors',  key: 'errors',  sortable: true, width: '40px' })
      if (this.showBlocked) headers.push({ title: 'Blocked', key: 'blocked', sortable: true, width: '40px' })
      return headers
    },
    pathHeaders() {
      const headers = [
        { title: 'Method',   key: 'method',   sortable: true,  width: '60px' },
        { title: 'Hostname', key: 'hostname', sortable: true },
        { title: 'Path',     key: 'path',     sortable: true }
      ]
      if (this.showSuccess) headers.push({ title: 'Success', key: 'pathSuccess', sortable: false, width: '40px' })
      if (this.showErrors)  headers.push({ title: 'Errors',  key: 'errors',      sortable: true,  width: '40px' })
      if (this.showBlocked) headers.push({ title: 'Blocked', key: 'pathBlocked', sortable: false, width: '40px' })
      return headers
    },
    filteredIPData() {
      return this.ipData.filter(ip =>
        (ip.success > 0 && this.showSuccess) ||
        (ip.errors  > 0 && this.showErrors)  ||
        (ip.blocked > 0 && this.showBlocked)
      )
    },
    filteredPathData() {
      return this.pathData.filter(p => {
        if (p.blocked) return this.showBlocked
        const successCount = p.count - p.errors
        return (successCount > 0 && this.showSuccess) || (p.errors > 0 && this.showErrors)
      })
    },
    highlightLat() {
      if (this.selectedLocation) return this.selectedLocation.lat
      if (!this.selectedIP) return null
      const entry = this.ipData.find(d => d.ip === this.selectedIP)
      return entry ? entry.lat : null
    },
    highlightLon() {
      if (this.selectedLocation) return this.selectedLocation.lon
      if (!this.selectedIP) return null
      const entry = this.ipData.find(d => d.ip === this.selectedIP)
      return entry ? entry.lon : null
    },
    rowProps() {
      return ({ item }) => ({
        class: (item.ip === this.selectedIP || this.isLocationSelected(item)) ? 'bg-primary-lighten-5' : '',
        style: 'cursor: pointer'
      })
    }
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
    isLocationSelected(item) {
      return this.selectedLocation !== null &&
        item.city === this.selectedLocation.city &&
        item.country === this.selectedLocation.country
    },
    toggleIP(ip) {
      this.selectedIP = this.selectedIP === ip ? '' : ip
      this.selectedLocation = null
      this.loadData()
    },
    toggleLocation(lat, lon, city, country) {
      if (this.selectedLocation?.city === city && this.selectedLocation?.country === country) {
        this.selectedLocation = null
      } else {
        this.selectedLocation = { lat, lon, city, country }
        this.selectedIP = ''
      }
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
      const filter = this.selectedIP
        ? `&ip=${encodeURIComponent(this.selectedIP)}`
        : (this.selectedLocation ? `&city=${encodeURIComponent(this.selectedLocation.city)}&country=${encodeURIComponent(this.selectedLocation.country)}` : '')

      const [map, ts, paths, ips] = await Promise.all([
        apiGet(`metrics/map?from=${from}&to=${to}${hn}`).catch(() => []),
        apiGet(`metrics/timeseries?from=${from}&to=${to}&granularity=${this.granularity}${hn}${filter}`).catch(() => []),
        apiGet(`metrics/paths?from=${from}&to=${to}${hn}${filter}`).catch(() => []),
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

<style scoped>
.compact-table :deep(th),
.compact-table :deep(td) {
  padding-left: 6px !important;
  padding-right: 6px !important;
}

/* Top IPs: card view by default, table above 2000px */
.ip-compact { display: block; }
.ip-table   { display: none; }
@media (min-width: 2000px) {
  .ip-compact { display: none; }
  .ip-table   { display: block; }
}

/* Top Paths: card view by default, table above 1100px */
.path-compact { display: block; }
.path-table   { display: none; }
@media (min-width: 1100px) {
  .path-compact { display: none; }
  .path-table   { display: block; }
}
</style>
