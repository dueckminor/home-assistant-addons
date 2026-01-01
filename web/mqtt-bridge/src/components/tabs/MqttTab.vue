<template>
  <div>
    <!-- Header -->
    <div class="d-flex align-center mb-4">
      <v-icon class="me-2" color="primary">mdi-network-outline</v-icon>
      <div>
        <h3 class="text-h6">MQTT Bridge Events</h3>
        <p class="text-body-2 text-medium-emphasis ma-0">Real-time MQTT message monitoring</p>
      </div>
      <v-spacer></v-spacer>
      <v-chip 
        :color="connectionStatus === 'connected' ? 'success' : connectionStatus === 'connecting' ? 'warning' : 'error'"
        variant="outlined"
        class="me-2"
      >
        <v-icon start>
          {{ connectionStatus === 'connected' ? 'mdi-lan-connect' : 
             connectionStatus === 'connecting' ? 'mdi-loading mdi-spin' : 'mdi-lan-disconnect' }}
        </v-icon>
        {{ connectionStatus.charAt(0).toUpperCase() + connectionStatus.slice(1) }}
      </v-chip>
      <v-btn
        color="error" 
        variant="outlined"
        @click="clearEvents"
        :disabled="events.length === 0"
      >
        <v-icon start>mdi-delete-sweep</v-icon>
        Clear
      </v-btn>
    </div>

    <!-- Events Table -->
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="d-flex align-center">
            <v-icon class="me-2">mdi-message-text-outline</v-icon>
            MQTT Events
            <v-spacer></v-spacer>
            <v-chip color="primary" variant="outlined">
              {{ events.length }} / 1000 events
            </v-chip>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <div class="events-container" ref="eventsContainer">
              <v-data-table
                :headers="headers"
                :items="events"
                :items-per-page="-1"
                class="elevation-0"
                hide-default-footer
                fixed-header
                height="500px"
              >
                <template v-slot:item.time="{ item }">
                  <span class="text-caption font-weight-bold">
                    {{ formatTime(item.time) }}
                  </span>
                </template>
                
                <template v-slot:item.topic="{ item }">
                  <code class="topic-code">{{ item.topic }}</code>
                </template>
                
                <template v-slot:item.value="{ item }">
                  <div class="value-cell">
                    <span v-if="isJsonValue(item.value)" class="json-value">
                      <v-btn
                        size="x-small"
                        variant="text"
                        @click="toggleJsonExpanded(item)"
                        class="me-1"
                      >
                        <v-icon size="12">
                          {{ item.jsonExpanded ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
                        </v-icon>
                      </v-btn>
                      <span v-if="!item.jsonExpanded" class="text-truncate">
                        {{ truncateValue(item.value) }}
                      </span>
                      <pre v-else class="json-pre">{{ formatJson(item.value) }}</pre>
                    </span>
                    <span v-else class="text-truncate">{{ truncateValue(item.value) }}</span>
                  </div>
                </template>
                
                <template v-slot:no-data>
                  <div class="text-center pa-4">
                    <v-icon size="48" color="grey-lighten-2" class="mb-2">mdi-message-outline</v-icon>
                    <p class="text-body-2 text-medium-emphasis">
                      {{ connectionStatus === 'connected' ? 'Waiting for MQTT events...' : 'Connect to view MQTT events' }}
                    </p>
                  </div>
                </template>
              </v-data-table>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script>
export default {
  name: 'MqttTab',
  data() {
    return {
      websocket: null,
      connectionStatus: 'disconnected', // 'disconnected', 'connecting', 'connected'
      events: [],
      maxEvents: 1000,
      headers: [
        {
          title: 'Time',
          key: 'time',
          width: '150px',
          sortable: false
        },
        {
          title: 'Topic',
          key: 'topic',
          width: '450px',
          sortable: false
        },
        {
          title: 'Value',
          key: 'value',
          width: 'auto',
          sortable: false
        }
      ],
      reconnectAttempts: 0,
      maxReconnectAttempts: 5,
      reconnectInterval: null
    }
  },
  mounted() {
    this.connectWebSocket()
  },
  beforeUnmount() {
    this.disconnectWebSocket()
  },
  methods: {
    connectWebSocket() {
      if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
        return
      }

      this.connectionStatus = 'connecting'
      
      // Determine WebSocket URL
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      const wsUrl = `${protocol}//${host}/api/events`

      try {
        this.websocket = new WebSocket(wsUrl)

        this.websocket.onopen = () => {
          console.log('WebSocket connected to:', wsUrl)
          this.connectionStatus = 'connected'
          this.reconnectAttempts = 0
          
          // Clear any existing reconnect interval
          if (this.reconnectInterval) {
            clearInterval(this.reconnectInterval)
            this.reconnectInterval = null
          }
        }

        this.websocket.onmessage = (event) => {
          try {
            const mqttEvent = JSON.parse(event.data)
            this.addEvent(mqttEvent)
          } catch (error) {
            console.error('Error parsing WebSocket message:', error)
          }
        }

        this.websocket.onclose = (event) => {
          console.log('WebSocket disconnected:', {
            code: event.code,
            reason: event.reason,
            wasClean: event.wasClean,
            timestamp: new Date().toISOString()
          })
          this.connectionStatus = 'disconnected'
          this.websocket = null
          
          // Auto-reconnect if not intentionally closed
          if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect()
          }
        }

        this.websocket.onerror = (error) => {
          console.error('WebSocket error:', error)
          this.connectionStatus = 'disconnected'
        }

      } catch (error) {
        console.error('Error creating WebSocket:', error)
        this.connectionStatus = 'disconnected'
        this.scheduleReconnect()
      }
    },

    disconnectWebSocket() {
      if (this.reconnectInterval) {
        clearInterval(this.reconnectInterval)
        this.reconnectInterval = null
      }
      
      if (this.websocket) {
        this.websocket.close(1000, 'Component unmounting')
        this.websocket = null
      }
      this.connectionStatus = 'disconnected'
    },

    scheduleReconnect() {
      if (this.reconnectInterval || this.reconnectAttempts >= this.maxReconnectAttempts) {
        return
      }

      this.reconnectAttempts++
      const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), 30000) // Exponential backoff, max 30s
      
      console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
      
      this.reconnectInterval = setTimeout(() => {
        this.reconnectInterval = null
        this.connectWebSocket()
      }, delay)
    },

    addEvent(mqttEvent) {
      // Add to beginning of array for newest-first display
      const event = {
        ...mqttEvent,
        id: Date.now() + Math.random(), // Unique ID for Vue's key
        jsonExpanded: false
      }
      
      this.events.unshift(event)
      
      // Keep only the most recent maxEvents
      if (this.events.length > this.maxEvents) {
        this.events = this.events.slice(0, this.maxEvents)
      }

      // Auto-scroll to top to show newest events
      this.$nextTick(() => {
        const container = this.$refs.eventsContainer
        if (container) {
          container.scrollTop = 0
        }
      })
    },

    clearEvents() {
      this.events = []
    },

    formatTime(timestamp) {
      const date = new Date(timestamp)
      return date.toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit',
        fractionalSecondDigits: 3
      })
    },

    isJsonValue(value) {
      if (typeof value !== 'string') return false
      try {
        const parsed = JSON.parse(value)
        return typeof parsed === 'object' && parsed !== null
      } catch {
        return false
      }
    },

    formatJson(value) {
      try {
        const parsed = JSON.parse(value)
        return JSON.stringify(parsed, null, 2)
      } catch {
        return value
      }
    },

    truncateValue(value) {
      const str = String(value)
      return str.length > 100 ? str.substring(0, 100) + '...' : str
    },

    toggleJsonExpanded(item) {
      item.jsonExpanded = !item.jsonExpanded
    }
  }
}
</script>

<style scoped>
.events-container {
  max-height: 500px;
  overflow-y: auto;
}

.topic-code {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  word-break: break-all;
}

.value-cell {
  max-width: 400px;
  font-family: 'Courier New', monospace;
}

.json-value {
  font-family: 'Courier New', monospace;
}

.json-pre {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  margin: 4px 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
}

.text-truncate {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  font-family: 'Courier New', monospace;
}

.mdi-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* Custom scrollbar for events container */
.events-container::-webkit-scrollbar {
  width: 8px;
}

.events-container::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.events-container::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 4px;
}

.events-container::-webkit-scrollbar-thumb:hover {
  background: #555;
}

/* Data table header styling */
:deep(.v-data-table__thead) {
  background-color: rgba(var(--v-theme-primary), 0.1);
}

:deep(.v-data-table-header__content) {
  font-weight: 600;
}

/* Row hover effect */
:deep(.v-data-table__tbody tr:hover) {
  background-color: rgba(var(--v-theme-primary), 0.05);
}
</style>