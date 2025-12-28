<template>
  <div class="timeline-container-wrapper">
    <div v-if="false" class="no-thumbnails">
      <v-alert type="info" class="ma-2">
        <v-icon>mdi-video-off</v-icon>
        No recordings found for {{ formatDate(selectedDate) }}. Use the day navigation buttons (◀◀ ▶▶) to browse other days.
      </v-alert>
    </div>
    
    <div class="timeline-container">
      <!-- Timeline with Navigation -->
      <div class="timeline-with-nav">
        <!-- Navigation buttons removed - now handled by NavigationControls component -->
        
        <!-- Main Timeline Bar -->
        <div 
          class="timeline-bar-container" 
          @click="onTimelineClick"
          @mousemove="onTimelineHover"
          @mouseleave="hideHoverPreview"
          @touchstart="onTouchStart"
          @touchmove="onTouchMove"
          @touchend="onTouchEnd"
        >
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
            ></div>
            
            <!-- Current time indicator -->
            <div
              v-if="selectedVideo"
              class="timeline-current-indicator"
              :style="getCurrentTimeStyle()"
            ></div>
          </div>
          
          <!-- Hover preview thumbnail -->
          <div 
            v-if="hoverPreview.visible && hoverPreview.thumbnail"
            class="timeline-hover-preview"
            :style="{
              left: `${hoverPreview.x}px`,
              top: `${hoverPreview.y}px`
            }"
          >
            <img 
              v-if="hoverPreview.thumbnail.hasThumbnail"
              :src="hoverPreview.thumbnail.url" 
              :alt="hoverPreview.thumbnail.name"
              class="hover-preview-image"
            />
            <div v-else class="hover-preview-placeholder">
              <div class="placeholder-video-icon">📹</div>
            </div>
            <div class="hover-preview-time">
              {{ formatTime(hoverPreview.thumbnail.timestamp) }}
            </div>
          </div>
          
          <!-- Timeline Header (below timeline) -->
          <div class="timeline-header">
            <div class="timeline-hours">
              <div 
                v-for="hour in 24" 
                :key="hour-1" 
                class="timeline-hour-marker"
                :style="{ left: `${((hour - 0.5) / 24) * 100}%` }"
              >
                {{ String(hour-1).padStart(2, '0') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Timeline',
  data() {
    return {
      hoverPreview: {
        visible: false,
        thumbnail: null,
        x: 0,
        y: 0
      },
      touchState: {
        startX: 0,
        startY: 0,
        hasMoved: false,
        startTime: 0
      }
    }
  },
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
    },
    canGoPrevious: {
      type: Boolean,
      default: false
    },
    canGoNext: {
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
    },
    
    onTimelineHover(event) {
      // Calculate hovered time based on position
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
        // Calculate thumbnail position, keeping it within screen bounds
        const thumbnailWidth = 208; // 200px + 4px padding on each side
        const containerWidth = rect.width;
        
        let thumbnailX = x - (thumbnailWidth / 2); // Center by default
        
        // Adjust if thumbnail would go off the left edge
        if (thumbnailX < 0) {
          thumbnailX = 0;
        }
        // Adjust if thumbnail would go off the right edge
        else if (thumbnailX + thumbnailWidth > containerWidth) {
          thumbnailX = containerWidth - thumbnailWidth;
        }
        
        // Position thumbnail above the timeline
        this.hoverPreview = {
          visible: true,
          thumbnail: closestVideo,
          x: thumbnailX,
          y: 0 // Let CSS transform handle the vertical positioning
        }
      }
    },
    
    hideHoverPreview() {
      this.hoverPreview.visible = false
    },
    
    onTouchStart(event) {
      // Track touch start position and time
      const touch = event.touches[0]
      this.touchState = {
        startX: touch.clientX,
        startY: touch.clientY,
        hasMoved: false,
        startTime: Date.now()
      }
      
      // Show preview immediately
      this.handleTouchHover(touch)
    },
    
    onTouchMove(event) {
      const touch = event.touches[0]
      const deltaX = Math.abs(touch.clientX - this.touchState.startX)
      const deltaY = Math.abs(touch.clientY - this.touchState.startY)
      
      // Only prevent default if we've actually moved (not just a tap)
      if (deltaX > 5 || deltaY > 5) {
        this.touchState.hasMoved = true
        event.preventDefault() // Prevent scrolling during preview
        this.handleTouchHover(touch)
      }
    },
    
    onTouchEnd(event) {
      const touchDuration = Date.now() - this.touchState.startTime
      
      // If it was a quick tap without movement, let the click event handle it
      if (!this.touchState.hasMoved && touchDuration < 200) {
        // Hide preview quickly for taps
        setTimeout(() => {
          this.hideHoverPreview()
        }, 100)
      } else {
        // Hide preview after delay for moves
        setTimeout(() => {
          this.hideHoverPreview()
        }, 300)
      }
      
      // Reset touch state
      this.touchState.hasMoved = false
    },
    
    handleTouchHover(touch) {
      // Calculate touched time based on position (similar to onTimelineHover)
      const rect = touch.target.closest('.timeline-bar-container').getBoundingClientRect()
      const x = touch.clientX - rect.left
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
        // Calculate thumbnail position, keeping it within screen bounds
        const thumbnailWidth = 208 // 200px + 4px padding on each side
        const containerWidth = rect.width
        
        let thumbnailX = x - (thumbnailWidth / 2) // Center by default
        
        // Adjust if thumbnail would go off the left edge
        if (thumbnailX < 0) {
          thumbnailX = 0
        }
        // Adjust if thumbnail would go off the right edge
        else if (thumbnailX + thumbnailWidth > containerWidth) {
          thumbnailX = containerWidth - thumbnailWidth
        }
        
        // Position thumbnail above the timeline
        this.hoverPreview = {
          visible: true,
          thumbnail: closestVideo,
          x: thumbnailX,
          y: 0 // Let CSS transform handle the vertical positioning
        }
      }
    }
  }
}
</script>

<style scoped>
.timeline-card {
  position: relative;
  overflow: visible; /* Allow hover preview to extend outside */
}

.no-thumbnails {
  text-align: center;
  padding: 20px;
}

.timeline-container {
  margin-top: 20px;
}

/* Timeline with navigation layout - simplified */
.timeline-with-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.timeline-bar-container {
  order: 1;
}

.day-nav-btn {
  opacity: 0.8;
  border-color: #666 !important;
}

.day-nav-btn:hover {
  opacity: 1;
  background-color: rgba(0, 0, 0, 0.04);
}

/* Landscape: buttons inline with timeline */
@media (orientation: landscape) {
  .timeline-with-nav {
    flex-direction: row;
    align-items: center;
    gap: 12px;
  }
  
  .timeline-nav-buttons {
    order: 0; /* Left side on landscape */
    flex-direction: row; /* Horizontal alignment */
    gap: 8px;
  }
  
  .timeline-bar-container {
    order: 1;
    flex: 1; /* Take remaining space */
  }
  
  .timeline-header {
    order: 2;
    margin-left: 0;
  }
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

/* Hide odd hour labels on small screens to reduce clutter */
@media (max-width: 768px) {
  .timeline-hour-marker:nth-child(even) {
    display: none;
  }
}

.timeline-bar-container {
  cursor: pointer;
  user-select: none;
  position: relative; /* For absolute positioning of hover preview */
  overflow: visible; /* Allow hover preview to extend outside */
}

/* Timeline hover preview */
.timeline-hover-preview {
  position: absolute;
  background: white;
  border-radius: 6px;
  padding: 4px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.3);
  pointer-events: none;
  z-index: 1000; /* High z-index to appear above all other elements */
  border: 2px solid #2196f3;
  /* Ensure it's not clipped by any parent containers */
  transform: translateY(-100%); /* Move fully above the hover point */
  margin-top: -10px; /* Additional spacing above timeline */
}

.hover-preview-image {
  width: 200px;
  height: 150px;
  object-fit: cover;
  border-radius: 3px;
  display: block;
}

.hover-preview-placeholder {
  width: 200px;
  height: 150px;
  background: #f0f0f0;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48px;
}

.hover-preview-time {
  position: absolute;
  bottom: 6px;
  left: 6px;
  right: 6px;
  background: rgba(0, 0, 0, 0.8);
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  text-align: center;
  border-radius: 2px;
  font-family: monospace;
}

.timeline-bar {
  position: relative;
  height: 40px;
  background: linear-gradient(135deg, #f5f5f5, #e8e8e8);
  border-radius: 8px;
  border: 1px solid #ddd;
  overflow: visible; /* Allow hover preview to extend outside */
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