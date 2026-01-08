<template>
  <div>
    <!-- Header -->
    <div class="d-flex align-center mb-4">
      <v-icon class="me-2" color="primary">mdi-console-line</v-icon>
      <div>
        <h3 class="text-h6">ESP Device Logs</h3>
        <p class="text-body-2 text-medium-emphasis ma-0">View debug logs from ESPHome devices</p>
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
    </div>

    <!-- Device Selection -->
    <v-row class="mb-4">
      <v-col cols="12" md="6">
        <v-select
          v-model="selectedDevice"
          :items="deviceOptions"
          label="Select ESP Device"
          variant="outlined"
          clearable
        >
          <template v-slot:item="{ props, item }">
            <v-list-item v-bind="props" :title="item.title" :subtitle="item.subtitle"></v-list-item>
          </template>
        </v-select>
      </v-col>
      <v-col cols="12" md="6">
        <v-switch
          v-model="autoScroll"
          label="Auto-scroll to latest"
          color="primary"
          class="me-4"
        ></v-switch>
      </v-col>
      <v-col cols="12" md="6" class="d-flex align-center justify-end">
        <v-btn
          v-if="selectedDevice"
          color="primary"
          variant="outlined"
          size="small"
          @click="scrollToBottomManual"
          class="me-2"
        >
          <v-icon start>mdi-arrow-down</v-icon>
          Scroll to Bottom
        </v-btn>
        <v-btn
          v-if="selectedDevice"
          color="warning"
          variant="outlined"
          size="small"
          @click="clearLogs"
        >
          <v-icon start>mdi-delete</v-icon>
          Clear Logs
        </v-btn>
      </v-col>
    </v-row>

    <!-- Log Display -->
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="d-flex align-center">
            <v-icon class="me-2">mdi-text-box-outline</v-icon>
            Debug Logs
            <v-spacer></v-spacer>
            <v-chip v-if="selectedDevice" color="primary" variant="outlined" class="me-2">
              {{ selectedDevice }}
            </v-chip>
            <v-chip v-if="selectedDevice && logHistory[selectedDevice]" color="success" variant="outlined">
              {{ logHistory[selectedDevice]?.length || 0 }} lines
            </v-chip>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <div v-if="!selectedDevice" class="text-center pa-8">
              <v-icon size="48" color="grey-lighten-2" class="mb-2">mdi-select-drag</v-icon>
              <p class="text-body-2 text-medium-emphasis">Select an ESP device to view its debug logs</p>
            </div>
            
            <div v-else-if="!debugLog" class="text-center pa-8">
              <v-icon size="48" color="grey-lighten-2" class="mb-2">mdi-text-box-outline</v-icon>
              <p class="text-body-2 text-medium-emphasis">No debug log available for {{ selectedDevice }}</p>
            </div>
            
            <v-card v-else class="ma-4" variant="outlined">
              <v-card-text class="pa-4">
                <pre class="log-content" v-html="debugLogHtml"></pre>
              </v-card-text>
            </v-card>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script>
import { useMqttTopics } from '../../composables/useMqttTopics.js'
import { computed, ref, watch, nextTick } from 'vue'

export default {
  name: 'EspLogsTab',
  setup() {
    const {
      topics,
      connectionStatus,
      getTopicsByPrefix
    } = useMqttTopics()

    const selectedDevice = ref(null)
    const autoScroll = ref(true)
    const logHistory = ref({}) // Store log history for each device
    const lastSeenValues = ref({}) // Track last seen value for each debug topic
    const MAX_LOG_SIZE = 1024 * 1024 // 1MB limit per device

    // ANSI color code mapping
    const ansiColors = {
      30: '#000000', 31: '#cd3131', 32: '#0dbc79', 33: '#e5e510', 
      34: '#2472c8', 35: '#bc3fbc', 36: '#11a8cd', 37: '#e5e5e5',
      90: '#666666', 91: '#f14c4c', 92: '#23d18b', 93: '#f5f543',
      94: '#3b8eea', 95: '#d670d6', 96: '#29b8db', 97: '#ffffff',
      // Background colors
      40: '#000000', 41: '#cd3131', 42: '#0dbc79', 43: '#e5e510',
      44: '#2472c8', 45: '#bc3fbc', 46: '#11a8cd', 47: '#e5e5e5',
      100: '#666666', 101: '#f14c4c', 102: '#23d18b', 103: '#f5f543',
      104: '#3b8eea', 105: '#d670d6', 106: '#29b8db', 107: '#ffffff'
    }

    // Convert ANSI escape codes to HTML
    const ansiToHtml = (text) => {
      if (!text) return ''
      
      // Replace ANSI escape sequences with HTML spans
      let html = text
      let currentStyles = []
      
      // Match ANSI escape sequences: ESC[...m
      const ansiRegex = /\x1b\[([0-9;]+)m/g
      let lastIndex = 0
      let result = ''
      let match
      
      while ((match = ansiRegex.exec(text)) !== null) {
        // Add text before this escape sequence
        result += escapeHtml(text.substring(lastIndex, match.index))
        
        // Parse the ANSI codes
        const codes = match[1].split(';').map(Number)
        
        codes.forEach(code => {
          if (code === 0) {
            // Reset all styles
            if (currentStyles.length > 0) {
              result += '</span>'
              currentStyles = []
            }
          } else if (code === 1) {
            // Bold
            currentStyles.push('font-weight: bold')
          } else if (code === 3) {
            // Italic
            currentStyles.push('font-style: italic')
          } else if (code === 4) {
            // Underline
            currentStyles.push('text-decoration: underline')
          } else if ((code >= 30 && code <= 37) || (code >= 90 && code <= 97)) {
            // Foreground color
            currentStyles.push(`color: ${ansiColors[code]}`)
          } else if ((code >= 40 && code <= 47) || (code >= 100 && code <= 107)) {
            // Background color
            currentStyles.push(`background-color: ${ansiColors[code]}`)
          }
        })
        
        // Apply styles if any
        if (currentStyles.length > 0) {
          result += `<span style="${currentStyles.join('; ')}">`
        }
        
        lastIndex = match.index + match[0].length
      }
      
      // Add remaining text
      result += escapeHtml(text.substring(lastIndex))
      
      // Close any open spans
      if (currentStyles.length > 0) {
        result += '</span>'
      }
      
      return result
    }
    
    // Escape HTML to prevent XSS
    const escapeHtml = (text) => {
      const div = document.createElement('div')
      div.textContent = text
      return div.innerHTML
    }

    // Find all ESP devices with debug topics
    const deviceOptions = computed(() => {
      const debugTopics = topics.value.filter(topic => topic.topic.endsWith('/debug'))
      return debugTopics.map(topic => {
        const deviceName = topic.topic.replace('/debug', '')
        const logLines = logHistory.value[deviceName]?.length || 0
        return {
          title: deviceName,
          subtitle: `${logLines} log lines`,
          value: deviceName
        }
      })
    })

    // Get the accumulated debug log for the selected device
    const debugLog = computed(() => {
      if (!selectedDevice.value || !logHistory.value[selectedDevice.value]) {
        return null
      }
      
      return logHistory.value[selectedDevice.value].join('\n')
    })
    
    // Get the HTML-formatted debug log with ANSI colors
    const debugLogHtml = computed(() => {
      if (!debugLog.value) return null
      return ansiToHtml(debugLog.value)
    })

    // Function to add log entry
    const addLogEntry = (deviceName, value) => {
      // Initialize log history for this device if it doesn't exist
      if (!logHistory.value[deviceName]) {
        logHistory.value[deviceName] = []
      }
      
      // Split the new value into lines and add each as a separate log entry
      const timestamp = new Date().toISOString()
      const logLines = value.split('\n').filter(line => line.trim())
      
      logLines.forEach(line => {
        if (line.trim()) {
          const logEntry = `[${timestamp}] ${line}`
          logHistory.value[deviceName].push(logEntry)
        }
      })
      
      // Limit log size per device (approximately 1MB)
      const currentLog = logHistory.value[deviceName].join('\n')
      if (currentLog.length > MAX_LOG_SIZE) {
        // Remove oldest entries until we're under the limit
        while (logHistory.value[deviceName].length > 0 && 
               logHistory.value[deviceName].join('\n').length > MAX_LOG_SIZE * 0.8) {
          logHistory.value[deviceName].shift()
        }
        // Add a truncation notice
        if (logHistory.value[deviceName][0] !== '[LOG TRUNCATED - Older entries removed to stay within 1MB limit]') {
          logHistory.value[deviceName].unshift('[LOG TRUNCATED - Older entries removed to stay within 1MB limit]')
        }
      }
      
      // Auto-scroll if enabled and this is the selected device
      if (autoScroll.value && selectedDevice.value === deviceName) {
        scrollToBottom()
      }
    }

    // Watch for changes in topics and accumulate log history
    watch(topics, (newTopics) => {
      const debugTopics = newTopics.filter(topic => topic.topic.endsWith('/debug'))
      
      debugTopics.forEach(topic => {
        const deviceName = topic.topic.replace('/debug', '')
        const newValue = topic.value
        
        if (!newValue) return
        
        // Get the last seen value for this device
        const lastValue = lastSeenValues.value[deviceName]
        
        // If this is the first time we see this topic, just store the value and initialize
        if (lastValue === undefined) {
          lastSeenValues.value[deviceName] = newValue
          // Initialize with current value
          addLogEntry(deviceName, newValue)
          return
        }
        
        // If the value changed, add new log lines
        if (newValue !== lastValue) {
          // Update the last seen value
          lastSeenValues.value[deviceName] = newValue
          
          // Add the new log entry
          addLogEntry(deviceName, newValue)
        }
      })
    }, { deep: true, immediate: true })

    // Auto-scroll function
    const scrollToBottom = () => {
      nextTick(() => {
        const logElement = document.querySelector('.log-content')
        if (logElement) {
          logElement.scrollTop = logElement.scrollHeight
        }
      })
    }

    // Clear logs for selected device
    const clearLogs = () => {
      if (selectedDevice.value && logHistory.value[selectedDevice.value]) {
        logHistory.value[selectedDevice.value] = []
      }
    }

    // Manual scroll to bottom
    const scrollToBottomManual = () => {
      scrollToBottom()
    }

    return {
      topics,
      connectionStatus,
      selectedDevice,
      autoScroll,
      deviceOptions,
      debugLog,
      debugLogHtml,
      clearLogs,
      scrollToBottomManual,
      logHistory,
      lastSeenValues
    }
  }
}
</script>

<style scoped>
.log-content {
  background-color: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 400px;
  overflow-y: auto;
  padding: 16px;
  border-radius: 4px;
}

.mdi-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>