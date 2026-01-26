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
      >
        No data gaps found! All measurements are complete.
      </v-alert>

      <div v-if="!loading && gapDays.length > 0">
        <v-alert
          type="info"
          variant="tonal"
          class="mb-4"
        >
          Found {{ totalGaps }} missing hours across {{ gapDays.length }} days
        </v-alert>

        <div class="d-flex justify-center align-center gap-2">
          <v-btn
            variant="outlined"
            @click="previousGap"
            :disabled="currentGapIndex === 0"
          >
            <v-icon>mdi-chevron-left</v-icon>
            Previous Gap
          </v-btn>
          <v-chip color="warning">
            {{ currentGapIndex + 1 }} / {{ gapDays.length }}
          </v-chip>
          <v-btn
            variant="outlined"
            @click="nextGap"
            :disabled="currentGapIndex === gapDays.length - 1"
          >
            Next Gap
            <v-icon>mdi-chevron-right</v-icon>
          </v-btn>
        </div>
      </div>

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
  emits: ['navigate-to-gap'],
  setup(props, { emit }) {
    const loading = ref(false)
    const gaps = ref([])
    const currentGapIndex = ref(0)
    
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

    const currentDay = computed(() => {
      if (gapDays.value.length === 0) return null
      return gapDays.value[currentGapIndex.value]
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
        currentGapIndex.value = 0 // Reset to first gap
      } catch (error) {
        console.error('Failed to load gaps:', error)
      } finally {
        loading.value = false
      }
    }

    const previousGap = () => {
      if (currentGapIndex.value > 0) {
        currentGapIndex.value--
        emit('navigate-to-gap', currentDay.value.date)
      }
    }

    const nextGap = () => {
      if (currentGapIndex.value < gapDays.value.length - 1) {
        currentGapIndex.value++
        emit('navigate-to-gap', currentDay.value.date)
      }
    }

    onMounted(() => {
      loadGaps()
    })

    return {
      loading,
      gapDays,
      currentGapIndex,
      totalGaps,
      loadGaps,
      previousGap,
      nextGap
    }
  }
}
</script>
