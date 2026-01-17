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

            <!-- Power Chart -->
            <v-card class="mb-4">
              <v-card-title>
                <v-icon class="me-2">mdi-chart-line</v-icon>
                Power Flow - Today
                <v-spacer></v-spacer>
                <v-btn
                  color="primary"
                  variant="outlined"
                  @click="refreshData"
                  :loading="loading"
                >
                  <v-icon start>mdi-refresh</v-icon>
                  Refresh
                </v-btn>
              </v-card-title>
              <v-card-text>
                <PowerChart :measurements="chartMeasurements" />
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
import GapsView from './components/GapsView.vue'
import { apiGet } from '../../shared/utils/homeassistant.js'

export default {
  name: 'App',
  components: {
    MetricCard,
    PowerChart,
    GapsView
  },
  setup() {
    const loading = ref(false)
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
    
    const keyMeasurements = [
      'from_grid',
      'battery_charge_from_grid',
      'to_grid',
      'solar_production',
      'battery_discharge',
      'battery_charge',
      'battery_soc'
    ]
    
    const chartMeasurements = computed(() => {
      return allMeasurements.value.filter(m => keyMeasurements.includes(m.name))
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
        
        // Get start of today
        const today = new Date()
        today.setHours(0, 0, 0, 0)
        const notBefore = today.toISOString()
        
        // Fetch measurements from API for today
        const keyMeasurementsQuery = `names=${keyMeasurements.join(',')}`
        const measurements = await apiGet(`measurements?${keyMeasurementsQuery}&not_before=${encodeURIComponent(notBefore)}`)
        allMeasurements.value = measurements
        
        // Fetch current measurements with previous for metrics
        const currentMeasurements = await apiGet('measurements?previous=true')
        
        connectionStatus.value.sensorCount = measurements.length
        
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
      refreshData,
      formatTime
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