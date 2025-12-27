<template>
  <v-card class="timeline-card ma-4" elevation="2">
    <v-card-text>
      <div v-if="thumbnails.length === 0 && !loading" class="no-thumbnails">
        <v-alert type="info" class="ma-2">
          No recordings found for {{ formatDate(selectedDate) }}
        </v-alert>
      </div>
      
      <div v-else class="timeline-container">
        <!-- Main Timeline Bar -->
        <div class="timeline-bar-container" @click="onTimelineClick">
          <div class="timeline-bar">
            <!-- Hour grid lines -->
            <div 
              v-for="hour in 24" 
              :key="`grid-${hour}`" 
              class="timeline-grid-line"
              :style="{ left: `${(hour / 24) * 100}%` }"
            ></div>
            
            <!-- Video blocks -->
            <div
              v-for="(video, index) in thumbnails"
              :key="video.videoName"
              class="timeline-video-block"
              :class="{ 
                active: selectedVideo && selectedVideo.name === video.videoName 
              }"
              :style="getVideoBlockStyle(video)"
              @click.stop="$emit('select-video', video)"
              :title="`${video.videoName} - ${formatTime(video.timestamp)}`"
            >
              <!-- Thumbnail preview on hover -->
              <div v-if="video.hasThumbnail" class="timeline-preview">
                <img :src="video.url" :alt="video.name" />
              </div>
            </div>
            
            <!-- Current time indicator -->
            <div
              v-if="selectedVideo"
              class="timeline-current-indicator"
              :style="getCurrentTimeStyle()"
            ></div>
          </div>
        </div>
        
        <!-- Timeline Header (below timeline) -->
        <div class="timeline-header">
          <div class="timeline-hours">
            <div 
              v-for="hour in 24" 
              :key="hour-1" 
              class="timeline-hour-marker"
            >
              {{ String(hour-1).padStart(2, '0') }}
            </div>
          </div>
        </div>
      </div>
    </v-card-text>
  </v-card>
</template>

<script>
export default {
  name: 'Timeline',
  props: {
    selectedVideo: {
      type: Object,
      default: null
    },
    thumbnails: {
      type: Array,
      default: () => []
    },
    selectedDate: {
      type: String,
      required: true
    },
    loading: {
      type: Boolean,
      default: false
    }
  },
  emits: ['select-video', 'timeline-click'],
  methods: {
    formatDate(dateString) {
      const date = new Date(dateString)
      return date.toLocaleDateString('en-US', {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      })
    },
    
    formatTime(timestamp) {
      const date = new Date(timestamp)
      return date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit'
      })
    },
    
    getVideoBlockStyle(video) {
      const date = new Date(video.timestamp)
      const hours = date.getHours()
      const minutes = date.getMinutes()
      const seconds = date.getSeconds()
      
      // Calculate position as percentage of the day
      const totalSeconds = hours * 3600 + minutes * 60 + seconds
      const daySeconds = 24 * 3600
      const leftPercent = (totalSeconds / daySeconds) * 100
      
      // Minimum width for visibility (about 5 minutes)
      const minWidthPercent = (300 / daySeconds) * 100 // 5 minutes in percentage
      
      return {
        left: `${leftPercent}%`,
        width: `${Math.max(minWidthPercent, 0.2)}%`
      }
    },
    
    getCurrentTimeStyle() {
      if (!this.selectedVideo) return {}
      
      const date = new Date(this.selectedVideo.timestamp)
      const hours = date.getHours()
      const minutes = date.getMinutes()
      const seconds = date.getSeconds()
      
      const totalSeconds = hours * 3600 + minutes * 60 + seconds
      const daySeconds = 24 * 3600
      const leftPercent = (totalSeconds / daySeconds) * 100
      
      return {
        left: `${leftPercent}%`
      }
    },
    
    onTimelineClick(event) {
      // Calculate clicked time based on position
      const rect = event.currentTarget.getBoundingClientRect()
      const x = event.clientX - rect.left
      const percentage = x / rect.width
      
      // Convert to time of day
      const totalSeconds = percentage * 24 * 3600
      const hours = Math.floor(totalSeconds / 3600)
      const minutes = Math.floor((totalSeconds % 3600) / 60)
      
      // Find closest video to this time
      const targetTime = new Date(this.selectedDate)
      targetTime.setHours(hours, minutes, 0, 0)
      
      let closestVideo = null
      let minDiff = Infinity
      
      this.thumbnails.forEach(video => {
        const diff = Math.abs(video.timestamp - targetTime.getTime())
        if (diff < minDiff) {
          minDiff = diff
          closestVideo = video
        }
      })
      
      if (closestVideo) {
        this.$emit('select-video', closestVideo)
      }
      
      this.$emit('timeline-click', { percentage, hours, minutes })
    }
  }
}
</script>

<style scoped>
.timeline-card {
  position: relative;
}

.no-thumbnails {
  text-align: center;
  padding: 20px;
}

.timeline-container {
  margin-top: 20px;
}

.timeline-header {
  margin-top: 8px;
  margin-bottom: 0;
}

.timeline-hours {
  display: flex;
  position: relative;
  height: 20px;
  margin-bottom: 4px;
}

.timeline-hour-marker {
  position: absolute;
  font-size: 12px;
  color: #666;
  transform: translateX(-50%);
  white-space: nowrap;
}

.timeline-hour-marker:nth-child(1) { left: 2.083%; }
.timeline-hour-marker:nth-child(2) { left: 6.25%; }
.timeline-hour-marker:nth-child(3) { left: 10.417%; }
.timeline-hour-marker:nth-child(4) { left: 14.583%; }
.timeline-hour-marker:nth-child(5) { left: 18.75%; }
.timeline-hour-marker:nth-child(6) { left: 22.917%; }
.timeline-hour-marker:nth-child(7) { left: 27.083%; }
.timeline-hour-marker:nth-child(8) { left: 31.25%; }
.timeline-hour-marker:nth-child(9) { left: 35.417%; }
.timeline-hour-marker:nth-child(10) { left: 39.583%; }
.timeline-hour-marker:nth-child(11) { left: 43.75%; }
.timeline-hour-marker:nth-child(12) { left: 47.917%; }
.timeline-hour-marker:nth-child(13) { left: 52.083%; }
.timeline-hour-marker:nth-child(14) { left: 56.25%; }
.timeline-hour-marker:nth-child(15) { left: 60.417%; }
.timeline-hour-marker:nth-child(16) { left: 64.583%; }
.timeline-hour-marker:nth-child(17) { left: 68.75%; }
.timeline-hour-marker:nth-child(18) { left: 72.917%; }
.timeline-hour-marker:nth-child(19) { left: 77.083%; }
.timeline-hour-marker:nth-child(20) { left: 81.25%; }
.timeline-hour-marker:nth-child(21) { left: 85.417%; }
.timeline-hour-marker:nth-child(22) { left: 89.583%; }
.timeline-hour-marker:nth-child(23) { left: 93.75%; }
.timeline-hour-marker:nth-child(24) { left: 97.917%; }

/* Hide odd hour labels on small screens to reduce clutter */
@media (max-width: 768px) {
  .timeline-hour-marker:nth-child(even) {
    display: none;
  }
}

.timeline-bar-container {
  cursor: pointer;
  user-select: none;
}

.timeline-bar {
  position: relative;
  height: 40px;
  background: linear-gradient(135deg, #f5f5f5, #e8e8e8);
  border-radius: 8px;
  border: 1px solid #ddd;
  overflow: hidden;
}

.timeline-grid-line {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: rgba(0, 0, 0, 0.1);
  pointer-events: none;
}

.timeline-grid-line:nth-child(13) {
  background: rgba(0, 0, 0, 0.3);
  width: 2px;
}

.timeline-current-indicator {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 3px;
  background: #ff5722;
  border-radius: 1px;
  z-index: 15;
  pointer-events: none;
  box-shadow: 0 0 6px rgba(255, 87, 34, 0.6);
}

.timeline-current-indicator::before {
  content: '';
  position: absolute;
  top: -6px;
  left: -3px;
  width: 9px;
  height: 9px;
  background: #ff5722;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.timeline-video-block {
  position: absolute;
  top: 2px;
  bottom: 2px;
  background: linear-gradient(135deg, #2196f3, #1976d2);
  border-radius: 2px;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 2px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}

.timeline-video-block:hover {
  background: linear-gradient(135deg, #42a5f5, #1e88e5);
  transform: scaleY(1.2);
  z-index: 10;
}

.timeline-video-block.active {
  background: linear-gradient(135deg, #ff5722, #d84315);
  box-shadow: 0 2px 8px rgba(255,87,34,0.4);
}

.timeline-preview {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-bottom: 8px;
  background: white;
  border-radius: 4px;
  padding: 4px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.3);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
  z-index: 20;
}

.timeline-video-block:hover .timeline-preview {
  opacity: 1;
}

.timeline-preview img {
  width: 80px;
  height: 60px;
  object-fit: cover;
  border-radius: 2px;
}
</style>