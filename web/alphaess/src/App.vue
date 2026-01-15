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

            <!-- Sensors Table -->
            <v-card>
              <v-card-title>
                <v-icon class="me-2">mdi-format-list-bulleted</v-icon>
                All Sensors
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
              <v-data-table
                :headers="sensorHeaders"
                :items="sensors"
                :loading="loading"
                item-value="name"
                class="elevation-1"
              >
                <template v-slot:item.value="{ item }">
                  <span class="font-weight-bold">{{ item.value }}</span>
                </template>
                <template v-slot:item.lastUpdate="{ item }">
                  {{ formatTime(item.lastUpdate) }}
                </template>
                <template v-slot:item.status="{ item }">
                  <v-chip
                    :color="item.status === 'online' ? 'success' : 'error'"
                    size="small"
                  >
                    {{ item.status }}
                  </v-chip>
                </template>
              </v-data-table>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import MetricCard from './components/MetricCard.vue'
import { apiGet } from './utils/api'

export default {
  name: 'App',
  components: {
    MetricCard
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
        unit: 'kW',
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
        unit: 'kW',
        icon: 'mdi-transmission-tower',
        color: 'blue'
      },
      {
        key: 'batteryPower',
        title: 'Battery Power',
        value: '0',
        unit: 'kW',
        icon: 'mdi-battery-charging',
        color: 'purple'
      }
    ])
    
    const sensors = ref([])
    
    const sensorHeaders = [
      { title: 'Sensor Name', key: 'name', sortable: true },
      { title: 'Value', key: 'value', sortable: false },
      { title: 'Unit', key: 'unit', sortable: false },
      { title: 'Status', key: 'status', sortable: true },
      { title: 'Last Update', key: 'lastUpdate', sortable: true }
    ]
    
    let refreshInterval = null
    
    const refreshData = async () => {
      loading.value = true
      try {
        // Simulate API calls - replace with actual API endpoints
        // const status = await apiGet('status')
        // const sensorData = await apiGet('sensors')
        
        // Mock data for demo
        connectionStatus.value = {
          connected: true,
          mqttConnected: true,
          alphaessConnected: true,
          mqttUri: 'tcp://core-mosquitto:1883',
          alphaessUri: 'tcp://192.168.1.100:502',
          lastUpdate: new Date(),
          sensorCount: 24
        }
        
        sensors.value = [
          {
            name: 'solar_production',
            value: '3.2',
            unit: 'kW',
            status: 'online',
            lastUpdate: new Date()
          },
          {
            name: 'battery_soc',
            value: '85',
            unit: '%',
            status: 'online',
            lastUpdate: new Date()
          },
          {
            name: 'grid_active_power',
            value: '1.5',
            unit: 'kW',
            status: 'online',
            lastUpdate: new Date()
          },
          {
            name: 'battery_power',
            value: '-0.8',
            unit: 'kW',
            status: 'online',
            lastUpdate: new Date()
          }
        ]
        
        // Update metrics
        metrics.value[0].value = '3.2'
        metrics.value[1].value = '85'
        metrics.value[2].value = '1.5'
        metrics.value[3].value = '-0.8'
        
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
      sensors,
      sensorHeaders,
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