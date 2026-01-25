<template>
  <v-app>
    <!-- Header -->
    <v-app-bar color="primary" dark elevation="2">
      <v-icon class="me-3">mdi-solar-power</v-icon>
      <v-toolbar-title>AlphaESS MQTT Bridge</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-chip 
        :color="connectionStatus.connected ? 'success' : 'error'" 
        variant="outlined"
      >
        <v-icon start>
          {{ connectionStatus.connected ? 'mdi-lan-connect' : 'mdi-lan-disconnect' }}
        </v-icon>
        {{ connectionStatus.connected ? 'Connected' : 'Disconnected' }}
      </v-chip>
    </v-app-bar>

    <!-- Main Content -->
    <v-main>
      <v-container fluid class="pa-4">
        <v-row>
          <v-col cols="12">
            <!-- Status Card -->
            <v-card class="mb-4">
              <v-card-title>
                <v-icon class="me-2">mdi-information</v-icon>
                System Status
              </v-card-title>
              <v-card-text>
                <v-row>
                  <v-col cols="12" md="6">
                    <v-list>
                      <v-list-item>
                        <v-list-item-title>MQTT Connection</v-list-item-title>
                        <v-list-item-subtitle>
                          {{ connectionStatus.mqttUri || 'Not configured' }}
                        </v-list-item-subtitle>
                        <template v-slot:append>
                          <v-icon 
                            :color="connectionStatus.mqttConnected ? 'success' : 'error'"
                          >
                            {{ connectionStatus.mqttConnected ? 'mdi-check-circle' : 'mdi-close-circle' }}
                          </v-icon>
                        </template>
                      </v-list-item>
                      <v-list-item>
                        <v-list-item-title>AlphaESS Connection</v-list-item-title>
                        <v-list-item-subtitle>
                          {{ connectionStatus.alphaessUri || 'Not configured' }}
                        </v-list-item-subtitle>
                        <template v-slot:append>
                          <v-icon 
                            :color="connectionStatus.alphaessConnected ? 'success' : 'error'"
                          >
                            {{ connectionStatus.alphaessConnected ? 'mdi-check-circle' : 'mdi-close-circle' }}
                          </v-icon>
                        </template>
                      </v-list-item>
                    </v-list>
                  </v-col>
                  <v-col cols="12" md="6">
                    <v-list>
                      <v-list-item>
                        <v-list-item-title>Last Update</v-list-item-title>
                        <v-list-item-subtitle>
                          {{ formatTime(connectionStatus.lastUpdate) }}
                        </v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <v-list-item-title>Active Sensors</v-list-item-title>
                        <v-list-item-subtitle>
                          {{ connectionStatus.sensorCount || 0 }} sensors
                        </v-list-item-subtitle>
                      </v-list-item>
                    </v-list>
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>

            <!-- Metrics Card -->
            <v-card class="mb-4">
              <v-card-title>
                <v-icon class="me-2">mdi-chart-line</v-icon>
                Current Metrics
              </v-card-title>
              <v-card-text>
                <v-row>
                  <v-col cols="12" sm="6" md="3" v-for="metric in metrics" :key="metric.key">
                    <MetricCard
                      :title="metric.title"
                      :value="metric.value"
                      :unit="metric.unit"
                      :icon="metric.icon"
                      :color="metric.color"
                    />
                  </v-col>
                </v-row>
              </v-card-text>
            </v-card>

            <!-- Power Flow / Battery State of Charge -->
            <v-card class="mb-4">
              <v-card-title>
                <v-icon class="me-2">mdi-chart-line</v-icon>
                Power Flow / Battery State of Charge
                <v-spacer></v-spacer>
                <div class="d-flex align-center ga-2">
                  <v-btn
                    icon="mdi-chevron-left"
                    size="small"
                    variant="text"
                    @click="previousDay"
                    :disabled="loading"
                  ></v-btn>
                  <v-menu
                    v-model="datePickerMenu"
                    :close-on-content-click="false"
                    location="bottom"
                  >
                    <template v-slot:activator="{ props }">
                      <v-btn
                        v-bind="props"
                        variant="outlined"
                        :disabled="loading"
                      >
                        <v-icon start>mdi-calendar</v-icon>
                        {{ formatDate(selectedDate) }}
                      </v-btn>
                    </template>
                    <v-date-picker
                      v-model="selectedDate"
                      @update:model-value="onDateSelected"
                      :max="today"
                    ></v-date-picker>
                  </v-menu>
                  <v-btn
                    icon="mdi-chevron-right"
                    size="small"
                    variant="text"
                    @click="nextDay"
                    :disabled="loading || isToday"
                  ></v-btn>
                  <v-btn
                    color="primary"
                    variant="outlined"
                    @click="refreshData"
                    :loading="loading"
                  >
                    <v-icon start>mdi-refresh</v-icon>
                    Refresh
                  </v-btn>
                </div>
              </v-card-title>
              <v-card-text>
                <SocChart :measurements="chartMeasurements" :selectedDate="selectedDate" />
                <PowerChart :measurements="chartMeasurements" :selectedDate="selectedDate" />
              </v-card-text>
            </v-card>

            <!-- Data Gaps -->
            <GapsView />
          </v-col>
        </v-row>
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import MetricCard from './components/MetricCard.vue'
import PowerChart from './components/PowerChart.vue'
import SocChart from './components/SocChart.vue'
import GapsView from './components/GapsView.vue'
import { apiGet } from '../../shared/utils/homeassistant.js'

export default {
  name: 'App',
  components: {
    MetricCard,
    PowerChart,
    SocChart,
    GapsView
  },
  setup() {
    const loading = ref(false)
    const selectedDate = ref(new Date())
    const datePickerMenu = ref(false)
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    
    const connectionStatus = ref({
      connected: false,
      mqttConnected: false,
      alphaessConnected: false,
      mqttUri: '',
      alphaessUri: '',
      lastUpdate: null,
      sensorCount: 0
    })
    
    const metrics = ref([
      {
        key: 'solarProduction',
        title: 'Solar Production',
        value: '0',
        unit: 'W',
        icon: 'mdi-solar-power',
        color: 'orange'
      },
      {
        key: 'batterySoc',
        title: 'Battery SOC',
        value: '0',
        unit: '%',
        icon: 'mdi-battery',
        color: 'green'
      },
      {
        key: 'gridPower',
        title: 'Grid Power',
        value: '0',
        unit: 'W',
        icon: 'mdi-transmission-tower',
        color: 'blue'
      },
      {
        key: 'batteryPower',
        title: 'Battery Power',
        value: '0',
        unit: 'W',
        icon: 'mdi-battery-charging',
        color: 'purple'
      }
    ])
    
    const allMeasurements = ref([])
    
    const chartMeasurements = computed(() => {
      // Return all aggregates directly (no filtering needed)
      return allMeasurements.value
    })
    
    let refreshInterval = null
    
    const refreshData = async () => {
      loading.value = true
      try {
        // Fetch status from API
        const status = await apiGet('status')
        connectionStatus.value = {
          connected: status.connected,
          mqttConnected: status.mqttConnected,
          alphaessConnected: status.alphaessConnected,
          mqttUri: status.mqttUri,
          alphaessUri: status.alphaessUri,
          lastUpdate: new Date(),
          sensorCount: 0
        }
        
        // Get start of selected day
        const startOfDay = new Date(selectedDate.value)
        startOfDay.setHours(0, 0, 0, 0)
        const from = startOfDay.toISOString()
        
        // Get start of next day (to include 23:00-24:00 hour)
        const nextDay = new Date(selectedDate.value)
        nextDay.setDate(nextDay.getDate() + 1)
        nextDay.setHours(0, 0, 0, 0)
        const to = nextDay.toISOString()
        
        // Get timezone
        const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
        
        // Fetch aggregated measurements from API for selected day
        const aggregates = await apiGet(`measurements/aggregate?interval=hourly&timezone=${encodeURIComponent(timezone)}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`)
        allMeasurements.value = aggregates
        
        // Fetch current measurements with previous for metrics
        const currentMeasurements = await apiGet('measurements?previous=true')
        
        connectionStatus.value.sensorCount = currentMeasurements.length
        
        // Helper to calculate power from accumulated energy (Wh to W)
        const calculatePower = (measurement) => {
          if (!measurement || !measurement.values || measurement.values.length < 2) {
            return measurement?.values?.[0]?.value || 0
          }
          const current = measurement.values[0]
          const previous = measurement.values[1]
          
          if (!current || !previous || !current.time || !previous.time) {
            return current?.value || 0
          }
          
          const energyDiff = current.value - previous.value // Wh
          const timeDiff = (new Date(current.time) - new Date(previous.time)) / 1000 / 3600 // hours
          
          if (timeDiff <= 0) return 0
          
          return energyDiff / timeDiff // W
        }
        
        // Helper to get latest value from measurement
        const getValue = (measurement) => {
          if (!measurement || !measurement.values || measurement.values.length === 0) {
            return 0
          }
          return measurement.values[0].value || 0
        }
        
        // Update metrics from current measurements
        const findMeasurement = (name) => currentMeasurements.find(m => m.name === name)
        
        const solarProduction = findMeasurement('solar_production') || findMeasurement('ppv')
        if (solarProduction) {
          const power = calculatePower(solarProduction)
          metrics.value[0].value = power.toFixed(0)
        }
        
        const batterySoc = findMeasurement('battery_soc') || findMeasurement('soc')
        if (batterySoc) {
          metrics.value[1].value = getValue(batterySoc).toFixed(0)
        }
        
        const gridPower = findMeasurement('grid_active_power') || findMeasurement('pgrid')
        if (gridPower) {
          metrics.value[2].value = getValue(gridPower).toFixed(0)
        }
        
        const batteryPower = findMeasurement('battery_power') || findMeasurement('pbat')
        if (batteryPower) {
          metrics.value[3].value = getValue(batteryPower).toFixed(0)
        }
        
      } catch (error) {
        console.error('Failed to refresh data:', error)
        connectionStatus.value.connected = false
      } finally {
        loading.value = false
      }
    }
    
    const formatTime = (date) => {
      if (!date) return 'Never'
      return new Date(date).toLocaleString()
    }
    
    const formatDate = (date) => {
      const d = new Date(date)
      const todayDate = new Date()
      todayDate.setHours(0, 0, 0, 0)
      const compareDate = new Date(d)
      compareDate.setHours(0, 0, 0, 0)
      
      if (compareDate.getTime() === todayDate.getTime()) {
        return 'Today'
      }
      
      const yesterday = new Date(todayDate)
      yesterday.setDate(yesterday.getDate() - 1)
      if (compareDate.getTime() === yesterday.getTime()) {
        return 'Yesterday'
      }
      
      return d.toLocaleDateString()
    }
    
    const isToday = computed(() => {
      const todayDate = new Date()
      todayDate.setHours(0, 0, 0, 0)
      const selected = new Date(selectedDate.value)
      selected.setHours(0, 0, 0, 0)
      return selected.getTime() >= todayDate.getTime()
    })
    
    const previousDay = () => {
      const newDate = new Date(selectedDate.value)
      newDate.setDate(newDate.getDate() - 1)
      selectedDate.value = newDate
      refreshData()
    }
    
    const nextDay = () => {
      const newDate = new Date(selectedDate.value)
      newDate.setDate(newDate.getDate() + 1)
      selectedDate.value = newDate
      refreshData()
    }
    
    const onDateSelected = () => {
      datePickerMenu.value = false
      refreshData()
    }
    
    onMounted(() => {
      refreshData()
      // Refresh every 30 seconds
      refreshInterval = setInterval(refreshData, 30000)
    })
    
    onUnmounted(() => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    })
    
    return {
      loading,
      connectionStatus,
      metrics,
      chartMeasurements,
      selectedDate,
      datePickerMenu,
      today,
      isToday,
      refreshData,
      formatTime,
      formatDate,
      previousDay,
      nextDay,
      onDateSelected
    }
  }
}
</script>

<style scoped>
.v-app-bar {
  z-index: 1000;
}

.v-main {
  padding-top: 64px;
}
</style>