<template>
  <!-- Thumbnail Strip with Navigation -->
  <div v-if="selectedVideo" class="video-thumbnail-navigation mt-3">
    <div class="thumbnail-nav-container">
      <v-btn
        v-if="canGoPrevious"
        icon="mdi-chevron-left"
        variant="text"
        size="large"
        @click="$emit('previous-video')"
        title="Previous Video (Shift + ←)"
        class="nav-btn-left"
      ></v-btn>
      
      <div class="thumbnail-strip-container">
        <div
          v-for="(thumbnail, index) in nearbyThumbnails"
          :key="`thumb-${thumbnail.videoName}-${index}`"
          class="timeline-thumbnail-item"
          :class="{ 
            'current-time': isCurrentTimeThumbnail(thumbnail)
          }"
          @click="$emit('select-video', thumbnail)"
        >
          <div class="timeline-thumbnail-wrapper">
            <img 
              v-if="thumbnail.hasThumbnail"
              :src="thumbnail.url" 
              :alt="thumbnail.name"
              class="timeline-thumbnail-image"
            />
            <div v-else class="timeline-thumbnail-placeholder">
              <div class="placeholder-video-icon">📹</div>
            </div>
            <div class="timeline-thumbnail-time">
              {{ formatTime(thumbnail.timestamp) }}
            </div>
          </div>
        </div>
      </div>
      
      <v-btn
        v-if="canGoNext"
        icon="mdi-chevron-right"
        variant="text"
        size="large"
        @click="$emit('next-video')"
        title="Next Video (Shift + →)"
        class="nav-btn-right"
      ></v-btn>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ThumbnailNavigation',
  props: {
    selectedVideo: {
      type: Object,
      default: null
    },
    thumbnails: {
      type: Array,
      default: () => []
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
  emits: ['select-video', 'previous-video', 'next-video'],
  computed: {
    nearbyThumbnails() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return []
      
      const currentIndex = this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
      
      if (currentIndex === -1) return []
      
      // Show 2 thumbnails before and 2 after the current one
      const start = Math.max(0, currentIndex - 2)
      const end = Math.min(this.thumbnails.length, currentIndex + 3)
      
      return this.thumbnails.slice(start, end)
    }
  },
  methods: {
    formatTime(timestamp) {
      const date = new Date(timestamp)
      return date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    },
    
    isCurrentTimeThumbnail(thumbnail) {
      return this.selectedVideo && thumbnail.videoName === this.selectedVideo.name
    }
  }
}
</script>

<style scoped>
/* Video Thumbnail Navigation Styles */
.video-thumbnail-navigation {
  margin: 16px 0;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.thumbnail-nav-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-btn-left,
.nav-btn-right {
  flex-shrink: 0;
}

.thumbnail-strip-container {
  display: flex;
  gap: 8px;
  justify-content: center;
  align-items: center;
  overflow-x: auto;
  padding: 4px 0;
  flex: 1;
  min-width: 0;
}

.timeline-thumbnail-item {
  flex-shrink: 0;
  cursor: pointer;
  transition: all 0.2s ease;
  border-radius: 4px;
  overflow: hidden;
}

.timeline-thumbnail-item:hover {
  transform: scale(1.05);
}

.timeline-thumbnail-item.current-time {
  box-shadow: 0 0 0 3px #2196f3;
  transform: scale(1.1);
}

.timeline-thumbnail-wrapper {
  position: relative;
  background: white;
  border-radius: 4px;
  overflow: hidden;
}

.timeline-thumbnail-image {
  width: 80px;
  height: 60px;
  object-fit: cover;
  display: block;
}

.timeline-thumbnail-placeholder {
  width: 80px;
  height: 60px;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.placeholder-video-icon {
  opacity: 0.6;
}

.timeline-thumbnail-time {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(0, 0, 0, 0.8);
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  text-align: center;
  font-family: monospace;
}
</style>