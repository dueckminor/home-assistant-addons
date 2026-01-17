<template>
  <v-card>
    <v-card-title>
      <v-icon class="me-2">mdi-calendar-remove</v-icon>
      Data Gaps
      <v-spacer></v-spacer>
      <v-btn
        color="primary"
        variant="outlined"
        @click="loadGaps"
        :loading="loading"
      >
        <v-icon start>mdi-refresh</v-icon>
        Refresh
      </v-btn>
    </v-card-title>
    
    <v-card-text>
      <v-alert
        v-if="!loading && gapDays.length === 0"
        type="success"
        variant="tonal"
        class="mb-4"
      >
        No data gaps found! All measurements are complete.
      </v-alert>

      <v-alert
        v-if="!loading && gapDays.length > 0"
        type="info"
        variant="tonal"
        class="mb-4"
      >
        Found {{ totalGaps }} missing hours across {{ gapDays.length }} days
      </v-alert>

      <v-expansion-panels v-if="gapDays.length > 0">
        <v-expansion-panel
          v-for="(day, index) in gapDays"
          :key="day.date"
        >
          <v-expansion-panel-title>
            <div class="d-flex align-center justify-space-between" style="width: 100%;">
              <div>
                <v-icon class="me-2" color="warning">mdi-calendar-alert</v-icon>
                <strong>{{ formatDate(day.date) }}</strong>
              </div>
              <v-chip
                size="small"
                color="warning"
                class="me-2"
              >
                {{ day.gaps.length }} missing hours
              </v-chip>
            </div>
          </v-expansion-panel-title>
          
          <v-expansion-panel-text>
            <v-list density="compact">
              <v-list-item
                v-for="gap in day.gaps"
                :key="gap.time"
              >
                <v-list-item-title>
                  <v-icon size="small" class="me-2">mdi-clock-alert-outline</v-icon>
                  {{ formatTime(gap.time) }}
                </v-list-item-title>
              </v-list-item>
            </v-list>
            
            <v-divider class="my-4"></v-divider>
            
            <div class="text-caption text-medium-emphasis">
              To fill these gaps, you can:
              <ul class="mt-2">
                <li>Download CSV data from AlphaESS for {{ formatDate(day.date) }}</li>
                <li>Import the data using the data import feature (coming soon)</li>
              </ul>
            </div>
          </v-expansion-panel-text>
        </v-expansion-panel>
      </v-expansion-panels>

      <v-progress-linear
        v-if="loading"
        indeterminate
        color="primary"
      ></v-progress-linear>
    </v-card-text>
  </v-card>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { apiGet } from '../../../shared/utils/homeassistant.js'

export default {
  name: 'GapsView',
  setup() {
    const loading = ref(false)
    const gaps = ref([])
    
    const keyMeasurements = [
      'from_grid',
      'battery_charge_from_grid',
      'to_grid',
      'solar_production',
      'battery_discharge',
      'battery_charge',
      'battery_soc'
    ]

    // Group gaps by date
    const gapDays = computed(() => {
      if (gaps.value.length === 0) return []
      
      // Use the first measurement to get gaps (they should all have the same gaps)
      const measurement = gaps.value[0]
      if (!measurement || !measurement.values) return []
      
      const dayMap = new Map()
      
      measurement.values.forEach(gap => {
        const date = new Date(gap.time)
        const dateKey = date.toISOString().split('T')[0]
        
        if (!dayMap.has(dateKey)) {
          dayMap.set(dateKey, {
            date: dateKey,
            gaps: []
          })
        }
        
        dayMap.get(dateKey).gaps.push(gap)
      })
      
      // Sort by date (newest first)
      return Array.from(dayMap.values()).sort((a, b) => 
        new Date(b.date) - new Date(a.date)
      )
    })

    const totalGaps = computed(() => {
      return gapDays.value.reduce((sum, day) => sum + day.gaps.length, 0)
    })

    const loadGaps = async () => {
      loading.value = true
      try {
        const keyMeasurementsQuery = `names=${keyMeasurements.join(',')}`
        const result = await apiGet(`gaps?${keyMeasurementsQuery}`)
        gaps.value = result
      } catch (error) {
        console.error('Failed to load gaps:', error)
      } finally {
        loading.value = false
      }
    }

    const formatDate = (dateStr) => {
      const date = new Date(dateStr)
      return date.toLocaleDateString('en-US', {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      })
    }

    const formatTime = (timeStr) => {
      const date = new Date(timeStr)
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
      })
    }

    onMounted(() => {
      loadGaps()
    })

    return {
      loading,
      gapDays,
      totalGaps,
      loadGaps,
      formatDate,
      formatTime
    }
  }
}
</script>

<style scoped>
.v-expansion-panel-title {
  padding: 12px 16px;
}
</style>
