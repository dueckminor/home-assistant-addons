<template>
  <div>
    <!-- Header -->
    <div class="d-flex align-center mb-4">
      <v-icon class="me-2" color="primary">mdi-network-outline</v-icon>
      <div>
        <h3 class="text-h6">MQTT Topics</h3>
        <p class="text-body-2 text-medium-emphasis ma-0">Monitor all MQTT topics with real-time updates</p>
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
        color="primary" 
        variant="outlined"
        @click="refreshTopics"
        :loading="loading"
        class="me-2"
      >
        <v-icon start>mdi-refresh</v-icon>
        Refresh
      </v-btn>
    </div>

    <!-- Topics Table -->
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="d-flex align-center">
            <v-icon class="me-2">mdi-message-text-outline</v-icon>
            MQTT Topics
            <v-spacer></v-spacer>
            <v-chip color="primary" variant="outlined">
              {{ topics.length }} topics
            </v-chip>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <v-data-table
              :headers="headers"
              :items="sortedTopics"
              :loading="loading"
              class="elevation-0"
              :items-per-page="25"
              :items-per-page-options="[25, 50, 100, -1]"
            >
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

              <template v-slot:item.lastUpdate="{ item }">
                <span class="text-caption">
                  {{ formatTime(item.lastUpdate) }}
                </span>
              </template>
              
              <template v-slot:no-data>
                <div class="text-center pa-4">
                  <v-icon size="48" color="grey-lighten-2" class="mb-2">mdi-message-outline</v-icon>
                  <p class="text-body-2 text-medium-emphasis">
                    {{ loading ? 'Loading topics...' : 'No MQTT topics found' }}
                  </p>
                </div>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script>
import { useMqttTopics } from '../../composables/useMqttTopics.js'

export default {
  name: 'MqttTab',
  setup() {
    const {
      topics,
      sortedTopics,
      connectionStatus,
      loading,
      refreshTopics,
      isJsonValue,
      formatJson,
      formatTime,
      truncateValue,
      toggleJsonExpanded
    } = useMqttTopics()

    return {
      topics,
      sortedTopics,
      connectionStatus,
      loading,
      refreshTopics,
      isJsonValue,
      formatJson,
      formatTime,
      truncateValue,
      toggleJsonExpanded
    }
  },
  data() {
    return {
      headers: [
        {
          title: 'Topic',
          key: 'topic',
          width: '40%',
          sortable: true
        },
        {
          title: 'Value',
          key: 'value',
          width: '45%',
          sortable: false
        },
        {
          title: 'Last Updated',
          key: 'lastUpdate',
          width: '15%',
          sortable: true
        }
      ]
    }
  }
}
</script>

<style scoped>
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