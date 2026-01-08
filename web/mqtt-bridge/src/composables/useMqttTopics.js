import { ref, computed, onUnmounted } from 'vue'
import { getBaseUrl, getWebSocketUrl } from '../../../shared/utils/homeassistant.js'

// Shared state across all components using this composable
const websocket = ref(null)
const connectionStatus = ref('disconnected') // 'disconnected', 'connecting', 'connected'
const topics = ref([])
const loading = ref(false)
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
let reconnectInterval = null
let lastRestLoadTime = null

export function useMqttTopics() {
  const sortedTopics = computed(() => {
    return [...topics.value].sort((a, b) => a.topic.localeCompare(b.topic))
  })

  const topicsByPattern = computed(() => {
    return (pattern) => {
      const regex = new RegExp(pattern.replace(/\*/g, '.*'))
      return topics.value.filter(topic => regex.test(topic.topic))
    }
  })

  const getTopicValue = (topicName) => {
    const topic = topics.value.find(t => t.topic === topicName)
    return topic ? topic.value : null
  }

  const getTopicsByPrefix = (prefix) => {
    return topics.value.filter(topic => topic.topic.startsWith(prefix))
  }

  async function loadTopics() {
    loading.value = true
    try {
      const baseUrl = getBaseUrl()
      const response = await fetch(`${baseUrl}/api/topics`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      const topicsData = await response.json()
      
      // Store when we completed the REST load
      lastRestLoadTime = Date.now()
      
      topics.value = topicsData.map(topic => ({
        ...topic,
        id: topic.topic,
        jsonExpanded: false,
        lastUpdate: topic.lastUpdate || topic.time || lastRestLoadTime
      }))
      
      console.log(`Loaded ${topics.value.length} topics`)
    } catch (error) {
      console.error('Error loading topics:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  function connectWebSocket() {
    if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
      return
    }

    connectionStatus.value = 'connecting'
    
    // Get WebSocket URL with proper ingress path handling
    const wsUrl = getWebSocketUrl('/api/topics?stream=true')

    try {
      websocket.value = new WebSocket(wsUrl)

      websocket.value.onopen = () => {
        console.log('WebSocket connected to:', wsUrl)
        connectionStatus.value = 'connected'
        reconnectAttempts.value = 0
        
        // Clear any existing reconnect interval
        if (reconnectInterval) {
          clearInterval(reconnectInterval)
          reconnectInterval = null
        }
        
        // Load topics after WebSocket is ready
        loadTopics()
      }

      websocket.value.onmessage = (event) => {
        try {
          const topicUpdate = JSON.parse(event.data)
          updateTopic(topicUpdate)
        } catch (error) {
          console.error('Error parsing WebSocket message:', error)
        }
      }

      websocket.value.onclose = (event) => {
        console.log('WebSocket disconnected:', {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
          timestamp: new Date().toISOString()
        })
        connectionStatus.value = 'disconnected'
        websocket.value = null
        
        // Auto-reconnect if not intentionally closed
        if (event.code !== 1000 && reconnectAttempts.value < maxReconnectAttempts) {
          scheduleReconnect()
        }
      }

      websocket.value.onerror = (error) => {
        console.error('WebSocket error:', error)
        connectionStatus.value = 'disconnected'
      }

    } catch (error) {
      console.error('Error creating WebSocket:', error)
      connectionStatus.value = 'disconnected'
      scheduleReconnect()
    }
  }

  function disconnectWebSocket() {
    if (reconnectInterval) {
      clearInterval(reconnectInterval)
      reconnectInterval = null
    }
    
    if (websocket.value) {
      websocket.value.close(1000, 'Component unmounting')
      websocket.value = null
    }
    connectionStatus.value = 'disconnected'
  }

  function scheduleReconnect() {
    if (reconnectInterval || reconnectAttempts.value >= maxReconnectAttempts) {
      return
    }

    reconnectAttempts.value++
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value - 1), 30000) // Exponential backoff, max 30s
    
    console.log(`Attempting to reconnect in ${delay}ms (attempt ${reconnectAttempts.value}/${maxReconnectAttempts})`)
    
    reconnectInterval = setTimeout(() => {
      reconnectInterval = null
      connectWebSocket()
    }, delay)
  }

  function updateTopic(topicData) {
    const updateTimestamp = topicData.lastUpdate || topicData.time || Date.now()
    
    // Ignore WebSocket updates that are older than our last REST load
    if (lastRestLoadTime && updateTimestamp < lastRestLoadTime) {
      console.log(`Ignoring stale WebSocket update for ${topicData.topic}: ${updateTimestamp} < ${lastRestLoadTime}`)
      return
    }
    
    const existingIndex = topics.value.findIndex(t => t.topic === topicData.topic)
    
    if (existingIndex >= 0) {
      // Update existing topic, preserve expanded state
      const existing = topics.value[existingIndex]
      topics.value.splice(existingIndex, 1, {
        ...topicData,
        id: topicData.topic,
        jsonExpanded: existing.jsonExpanded,
        lastUpdate: updateTimestamp
      })
    } else {
      // Add new topic
      topics.value.push({
        ...topicData,
        id: topicData.topic,
        jsonExpanded: false,
        lastUpdate: updateTimestamp
      })
    }
  }

  async function refreshTopics() {
    await loadTopics()
  }

  function initializeMqttConnection() {
    // Only connect if not already connected or connecting
    if (connectionStatus.value === 'disconnected') {
      connectWebSocket()
    }
  }

  // Helper functions for common patterns
  function isJsonValue(value) {
    if (typeof value !== 'string') return false
    try {
      const parsed = JSON.parse(value)
      return typeof parsed === 'object' && parsed !== null
    } catch {
      return false
    }
  }

  function formatJson(value) {
    try {
      const parsed = JSON.parse(value)
      return JSON.stringify(parsed, null, 2)
    } catch {
      return value
    }
  }

  function formatTime(timestamp) {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit'
    })
  }

  function truncateValue(value, maxLength = 100) {
    const str = String(value)
    return str.length > maxLength ? str.substring(0, maxLength) + '...' : str
  }

  function toggleJsonExpanded(item) {
    item.jsonExpanded = !item.jsonExpanded
  }

  // Cleanup function - call this when the last component using this composable unmounts
  let componentCount = 0
  
  function incrementComponentCount() {
    componentCount++
    if (componentCount === 1) {
      // First component using this composable
      initializeMqttConnection()
    }
  }

  function decrementComponentCount() {
    componentCount--
    if (componentCount === 0) {
      // Last component stopped using this composable
      disconnectWebSocket()
    }
  }

  // Auto-cleanup when component unmounts
  onUnmounted(() => {
    decrementComponentCount()
  })

  // Initialize connection when first component mounts
  incrementComponentCount()

  return {
    // State
    topics: topics,
    sortedTopics,
    connectionStatus,
    loading,
    
    // Topic access methods
    getTopicValue,
    getTopicsByPrefix,
    topicsByPattern,
    
    // Connection methods
    initializeMqttConnection,
    refreshTopics,
    
    // Helper methods
    isJsonValue,
    formatJson,
    formatTime,
    truncateValue,
    toggleJsonExpanded,
    
    // Internal methods (exposed for advanced usage)
    connectWebSocket,
    disconnectWebSocket,
    loadTopics
  }
}