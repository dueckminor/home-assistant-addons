<template>
  <v-app>
    <v-app-bar app color="primary" dark>
      <v-toolbar-title>
        <v-icon class="mr-2">mdi-shield-check</v-icon>
        Security Camera System
      </v-toolbar-title>
      
      <v-spacer></v-spacer>
      
      <!-- Day Navigation -->
      <v-btn
        variant="text"
        icon="mdi-chevron-left"
        @click="previousDay"
        :disabled="loading"
      ></v-btn>
      
      <v-chip class="mx-2" color="secondary">
        {{ formatDate(selectedDate) }}
      </v-chip>
      
      <v-btn
        variant="text"
        icon="mdi-chevron-right"
        @click="nextDay"
        :disabled="loading || isToday"
      ></v-btn>
      
      <v-menu offset-y>
        <template v-slot:activator="{ props }">
          <v-btn variant="text" icon="mdi-calendar" v-bind="props" class="ml-2"></v-btn>
        </template>
        <v-date-picker
          v-model="selectedDate"
          @update:model-value="loadDayFiles"
        ></v-date-picker>
      </v-menu>
    </v-app-bar>

    <v-main>
      <div class="video-container">
        <!-- Video Player Section -->
        <v-card class="video-card ma-4" elevation="2">
          <v-card-text>
            <div v-if="selectedVideo" class="video-player-wrapper">
              <div class="video-container-with-controls">
                <video
                  ref="videoPlayer"
                  :src="selectedVideo.url"
                  controls
                  autoplay
                  muted
                  preload="metadata"
                  class="video-player"
                  @loadedmetadata="onVideoLoaded"
                  @error="onVideoError"
                  @ended="onVideoEnded"
                >
                  Your browser does not support the video tag.
                </video>
                
                <!-- Video Overlay Controls -->
                <div 
                  class="video-overlay-controls" 
                  :class="{ 'fullscreen-controls': isFullscreen }"
                  @mousemove="showFullscreenControls"
                  @mouseleave="hideFullscreenControls"
                >
                  <button
                    v-if="canGoPrevious"
                    class="video-nav-btn video-nav-prev"
                    @click="playPreviousVideo"
                    title="Previous Video (Shift + ←)"
                  >
                    <v-icon size="32">mdi-skip-previous</v-icon>
                  </button>
                  
                  <button
                    v-if="canGoNext"
                    class="video-nav-btn video-nav-next"
                    @click="playNextVideo"
                    title="Next Video (Shift + →)"
                  >
                    <v-icon size="32">mdi-skip-next</v-icon>
                  </button>
                </div>
              </div>
              <div class="video-info mt-2">
                <v-chip color="primary" small>
                  {{ selectedVideo.name }}
                </v-chip>
                <v-chip color="secondary" small class="ml-2">
                  {{ formatDateTime(selectedVideo.timestamp) }}
                </v-chip>
                <v-chip 
                  v-if="selectedVideo.hasThumbnail && selectedVideo.timeDiffMinutes > 0" 
                  color="warning" 
                  small 
                  class="ml-2"
                >
                  Thumbnail ±{{ selectedVideo.timeDiffMinutes }}min
                </v-chip>
                <v-chip 
                  v-else-if="!selectedVideo.hasThumbnail" 
                  color="info" 
                  small 
                  class="ml-2"
                >
                  No thumbnail
                </v-chip>
              </div>
            </div>
            <div v-else class="no-video-placeholder">
              <v-icon size="64" color="grey lighten-2">mdi-video-off</v-icon>
              <p class="text-h6 grey--text mt-4">Select a thumbnail from the timeline below</p>
            </div>
          </v-card-text>
        </v-card>

        <!-- Loading indicator -->
        <v-progress-linear
          v-if="loading"
          indeterminate
          color="primary"
          class="ma-4"
        ></v-progress-linear>

        <!-- Timeline Section -->
        <v-card class="timeline-card ma-4" elevation="2">
          <v-card-title>
            <v-icon class="mr-2">mdi-timeline</v-icon>
            Timeline - {{ formatDate(selectedDate) }}
            <v-spacer></v-spacer>
            <v-btn
              @click="refreshFiles"
              :loading="loading"
              color="primary"
              small
            >
              <v-icon small>mdi-refresh</v-icon>
              Refresh
            </v-btn>
          </v-card-title>
          
          <v-card-text>
            <div v-if="thumbnails.length === 0 && !loading" class="no-thumbnails">
              <v-alert type="info" class="ma-2">
                No recordings found for {{ formatDate(selectedDate) }}
              </v-alert>
            </div>
            
            <div v-else class="timeline-container">
              <div class="timeline-scroll">
                <div class="thumbnail-grid">
                  <div
                    v-for="thumbnail in thumbnails"
                    :key="thumbnail.videoName"
                    class="thumbnail-item"
                    :class="{ 
                      active: selectedVideo && selectedVideo.name === thumbnail.videoName,
                      'no-thumbnail': !thumbnail.hasThumbnail
                    }"
                    @click="selectVideo(thumbnail)"
                  >
                    <div class="thumbnail-wrapper">
                      <img
                        v-if="thumbnail.hasThumbnail"
                        :src="thumbnail.url"
                        :alt="thumbnail.name"
                        class="thumbnail-image"
                        @error="onThumbnailError"
                      />
                      <div v-else class="thumbnail-placeholder">
                        <v-icon size="24" color="grey lighten-1">mdi-video</v-icon>
                      </div>
                      <div class="thumbnail-overlay">
                        <v-icon class="play-icon">mdi-play-circle</v-icon>
                        <div class="timestamp">{{ formatTime(thumbnail.timestamp) }}</div>
                        <div v-if="thumbnail.hasThumbnail && thumbnail.timeDiffMinutes > 0" class="time-diff">
                          ±{{ thumbnail.timeDiffMinutes }}m
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </v-card-text>
        </v-card>
      </div>
    </v-main>
  </v-app>
</template>

<script>
export default {
  name: 'SecurityApp',
  data() {
    return {
      selectedDate: new Date().toISOString().substr(0, 10),
      selectedVideo: null,
      thumbnails: [],
      loading: false,
      baseUrl: this.getBaseUrl(),
      isFullscreen: false,
      fullscreenControlsTimeout: null,
    }
  },
  
  mounted() {
    this.loadDayFiles()
    // Add keyboard event listeners
    window.addEventListener('keydown', this.handleKeydown)
    // Add fullscreen change listeners
    document.addEventListener('fullscreenchange', this.onFullscreenChange)
    document.addEventListener('webkitfullscreenchange', this.onFullscreenChange)
    document.addEventListener('mozfullscreenchange', this.onFullscreenChange)
    document.addEventListener('msfullscreenchange', this.onFullscreenChange)
  },
  
  beforeUnmount() {
    // Clean up event listeners
    window.removeEventListener('keydown', this.handleKeydown)
    document.removeEventListener('fullscreenchange', this.onFullscreenChange)
    document.removeEventListener('webkitfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('mozfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('msfullscreenchange', this.onFullscreenChange)
  },
  
  computed: {
    isToday() {
      const today = new Date().toISOString().substr(0, 10)
      return this.selectedDate === today
    },
    
    currentVideoIndex() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return -1
      return this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
    },
    
    canGoPrevious() {
      return this.currentVideoIndex > 0
    },
    
    canGoNext() {
      return this.currentVideoIndex >= 0 && this.currentVideoIndex < this.thumbnails.length - 1
    }
  },
  
  methods: {
    getBaseUrl() {
      // For Home Assistant ingress, detect if we're running under a hassio_ingress path
      const path = window.location.pathname
      const ingressMatch = path.match(/^(\/api\/hassio_ingress\/[^\/]+)/)
      
      if (ingressMatch) {
        // Running under Home Assistant ingress
        return ingressMatch[1]
      } else {
        // Local development or direct access
        return ''
      }
    },
    
    previousDay() {
      const currentDate = new Date(this.selectedDate)
      currentDate.setDate(currentDate.getDate() - 1)
      this.selectedDate = currentDate.toISOString().substr(0, 10)
      this.loadDayFiles()
    },
    
    nextDay() {
      if (this.isToday) return // Prevent going beyond today
      
      const currentDate = new Date(this.selectedDate)
      currentDate.setDate(currentDate.getDate() + 1)
      this.selectedDate = currentDate.toISOString().substr(0, 10)
      this.loadDayFiles()
    },
    
    handleKeydown(event) {
      // Only handle arrow keys when not typing in an input field
      if (event.target.tagName.toLowerCase() === 'input') return
      
      if (event.key === 'ArrowLeft') {
        event.preventDefault()
        // Check if Shift is held for video navigation, otherwise day navigation
        if (event.shiftKey) {
          this.playPreviousVideo()
        } else {
          this.previousDay()
        }
      } else if (event.key === 'ArrowRight') {
        event.preventDefault()
        // Check if Shift is held for video navigation, otherwise day navigation
        if (event.shiftKey) {
          this.playNextVideo()
        } else {
          this.nextDay()
        }
      }
    },
    async loadDayFiles() {
      this.loading = true
      try {
        // Format date for API call (cam1/2025/12/25)
        const date = new Date(this.selectedDate)
        const year = date.getFullYear()
        const month = String(date.getMonth() + 1).padStart(2, '0')
        const day = String(date.getDate()).padStart(2, '0')
        
        const path = `cam1/${year}/${month}/${day}`
        
        const response = await fetch(`${this.baseUrl}/api/ftp/${path}`)
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`)
        }
        
        const data = await response.json()
        
        // Filter JPG and MP4 files separately
        const jpgFiles = data.files.filter(file => 
          file.name.toLowerCase().endsWith('.jpg') || 
          file.name.toLowerCase().endsWith('.jpeg')
        )
        
        const mp4Files = data.files.filter(file => 
          file.name.toLowerCase().endsWith('.mp4')
        )
        
        // Create video objects with timestamps
        const videos = mp4Files.map(mp4File => ({
          name: mp4File.name,
          path: mp4File.path,
          url: `${this.baseUrl}/api/files/${mp4File.path}`,
          timestamp: this.extractTimestampFromFilename(mp4File.name),
          size: mp4File.size,
          modTime: new Date(mp4File.modTime)
        }))
        
        // Create thumbnail objects and match them to closest videos
        this.thumbnails = videos.map(video => {
          // Find the closest JPG file by timestamp
          let closestJpg = null
          let minTimeDiff = Infinity
          
          jpgFiles.forEach(jpgFile => {
            const jpgTimestamp = this.extractTimestampFromFilename(jpgFile.name)
            const timeDiff = Math.abs(video.timestamp - jpgTimestamp)
            
            if (timeDiff < minTimeDiff) {
              minTimeDiff = timeDiff
              closestJpg = jpgFile
            }
          })
          
          return {
            name: closestJpg ? closestJpg.name : `thumb_${video.name}`,
            videoName: video.name,
            videoPath: video.path,
            url: closestJpg ? `${this.baseUrl}/api/files/${closestJpg.path}` : null,
            videoUrl: video.url,
            timestamp: video.timestamp,
            size: video.size,
            modTime: video.modTime,
            hasVideo: true,
            hasThumbnail: !!closestJpg,
            timeDiffMinutes: closestJpg ? Math.round(minTimeDiff / (1000 * 60)) : null // Time diff in minutes
          }
        })
        
        // Sort by timestamp
        this.thumbnails.sort((a, b) => a.timestamp - b.timestamp)
        
        // Auto-select first video if available
        if (this.thumbnails.length > 0) {
          this.selectVideo(this.thumbnails[0])
        } else {
          this.selectedVideo = null
        }
        
      } catch (error) {
        console.error('Error loading files:', error)
        this.thumbnails = []
        this.selectedVideo = null
      } finally {
        this.loading = false
      }
    },
    
    selectVideo(thumbnail) {
      this.selectedVideo = {
        name: thumbnail.videoName,
        url: thumbnail.videoUrl,
        thumbnail: thumbnail.url,
        timestamp: thumbnail.timestamp,
        hasThumbnail: thumbnail.hasThumbnail,
        timeDiffMinutes: thumbnail.timeDiffMinutes
      }
      
      // Scroll video into view and autoplay
      this.$nextTick(() => {
        if (this.$refs.videoPlayer) {
          this.$refs.videoPlayer.load()
          // Attempt autoplay after the video is loaded
          this.$refs.videoPlayer.addEventListener('loadeddata', () => {
            this.playVideo()
          }, { once: true })
        }
      })
    },
    
    playVideo() {
      if (this.$refs.videoPlayer) {
        const playPromise = this.$refs.videoPlayer.play()
        
        if (playPromise !== undefined) {
          playPromise.catch(error => {
            console.warn('Autoplay failed:', error)
            // Autoplay was prevented, user will need to click play manually
          })
        }
      }
    },
    
    extractTimestampFromFilename(filename) {
      // Extract timestamp from Reolink filename format: RH94 2_00_20251225210602.jpg
      const match = filename.match(/(\d{14})\.(jpg|jpeg|mp4)$/i)
      if (match) {
        const timestampStr = match[1] // "20251225210602"
        const year = parseInt(timestampStr.substr(0, 4))
        const month = parseInt(timestampStr.substr(4, 2)) - 1 // Month is 0-based
        const day = parseInt(timestampStr.substr(6, 2))
        const hour = parseInt(timestampStr.substr(8, 2))
        const minute = parseInt(timestampStr.substr(10, 2))
        const second = parseInt(timestampStr.substr(12, 2))
        
        return new Date(year, month, day, hour, minute, second)
      }
      
      // Fallback to file modification time
      return new Date()
    },
    
    formatDateTime(date) {
      return date.toLocaleString()
    },
    
    formatDate(dateStr) {
      return new Date(dateStr).toLocaleDateString()
    },
    
    formatTime(date) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    },
    
    refreshFiles() {
      this.loadDayFiles()
    },
    
    onVideoLoaded() {
      console.log('Video loaded successfully')
      // Additional autoplay attempt when video metadata is loaded
      this.playVideo()
    },
    
    onVideoError(event) {
      console.error('Video loading error:', event)
    },
    
    onVideoEnded() {
      console.log('Video ended, looking for next video')
      this.playNextVideo()
    },
    
    playNextVideo() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return
      
      // Find current video index in thumbnails
      const currentIndex = this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
      
      if (currentIndex === -1) return
      
      // Get next video (or loop to first if at the end)
      const nextIndex = currentIndex + 1
      if (nextIndex < this.thumbnails.length) {
        // Play next video
        this.selectVideo(this.thumbnails[nextIndex])
      } else {
        // Reached the end of videos
        console.log('Reached end of video timeline')
        // Optionally could loop back to first video:
        // this.selectVideo(this.thumbnails[0])
      }
    },
    
    playPreviousVideo() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return
      
      // Find current video index in thumbnails
      const currentIndex = this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
      
      if (currentIndex === -1) return
      
      // Get previous video
      const prevIndex = currentIndex - 1
      if (prevIndex >= 0) {
        // Play previous video
        this.selectVideo(this.thumbnails[prevIndex])
      } else {
        // At the beginning of videos
        console.log('At beginning of video timeline')
      }
    },
    
    onFullscreenChange() {
      this.isFullscreen = !!(
        document.fullscreenElement ||
        document.webkitFullscreenElement ||
        document.mozFullScreenElement ||
        document.msFullscreenElement
      )
    },
    
    showFullscreenControls() {
      if (this.isFullscreen) {
        // Clear any existing timeout
        if (this.fullscreenControlsTimeout) {
          clearTimeout(this.fullscreenControlsTimeout)
        }
      }
    },
    
    hideFullscreenControls() {
      if (this.isFullscreen) {
        // Hide controls after a delay in fullscreen
        this.fullscreenControlsTimeout = setTimeout(() => {
          // Controls will fade out automatically via CSS
        }, 3000)
      }
    },
    
    onThumbnailError(event) {
      console.error('Thumbnail loading error:', event)
      // Could set a placeholder image here
    }
  }
}
</script>

<style>
.video-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 64px);
}

.video-card {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.video-player-wrapper {
  display: flex;
  flex-direction: column;
  width: 100%;
  position: relative;
}

.video-container-with-controls {
  position: relative;
  width: 100%;
}

.video-player {
  width: 100%;
  height: 60vh;
  background: #000;
  object-fit: contain;
}

.video-overlay-controls {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  opacity: 0;
  transition: opacity 0.3s ease;
  z-index: 1000;
}

.video-container-with-controls:hover .video-overlay-controls,
.video-overlay-controls.fullscreen-controls {
  opacity: 1;
}

/* Fullscreen specific styles */
video:fullscreen + .video-overlay-controls,
video:-webkit-full-screen + .video-overlay-controls,
video:-moz-full-screen + .video-overlay-controls,
video:-ms-fullscreen + .video-overlay-controls {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2147483647;
}

video:fullscreen:hover + .video-overlay-controls,
video:-webkit-full-screen:hover + .video-overlay-controls,
video:-moz-full-screen:hover + .video-overlay-controls,
video:-ms-fullscreen:hover + .video-overlay-controls {
  opacity: 1;
}

.video-nav-btn {
  pointer-events: all;
  background: rgba(0, 0, 0, 0.7);
  border: none;
  border-radius: 50%;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  color: white;
}

.video-nav-btn:hover {
  background: rgba(0, 0, 0, 0.9);
  transform: scale(1.1);
}

.video-nav-btn:active {
  transform: scale(0.95);
}

.video-nav-prev {
  margin-left: 20px;
}

.video-nav-next {
  margin-right: 20px;
}

.no-video-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.video-info {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
}

.timeline-card {
  flex-shrink: 0;
  max-height: 300px;
}

.timeline-container {
  overflow: hidden;
}

.timeline-scroll {
  overflow-x: auto;
  overflow-y: hidden;
  padding-bottom: 8px;
}

.thumbnail-grid {
  display: flex;
  gap: 8px;
  padding: 8px 0;
}

.thumbnail-item {
  flex-shrink: 0;
  cursor: pointer;
  border-radius: 4px;
  overflow: hidden;
  transition: all 0.2s ease;
}

.thumbnail-item:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 8px rgba(0,0,0,0.3);
}

.thumbnail-item.active {
  border: 2px solid #1976d2;
  box-shadow: 0 4px 12px rgba(25,118,210,0.4);
}

.thumbnail-item.no-thumbnail {
  opacity: 0.8;
}

.thumbnail-placeholder {
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #f5f5f5, #e0e0e0);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed #ccc;
}

.thumbnail-wrapper {
  position: relative;
  width: 120px;
  height: 90px;
}

.thumbnail-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.thumbnail-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
  color: white;
  padding: 4px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.thumbnail-item:hover .thumbnail-overlay {
  opacity: 1;
}

.play-icon {
  font-size: 20px;
}

.no-video-icon {
  font-size: 20px;
  color: #ff5722;
}

.timestamp {
  font-size: 10px;
  font-weight: bold;
}

.time-diff {
  font-size: 9px;
  color: #ffeb3b;
  font-weight: bold;
}

.no-thumbnails {
  text-align: center;
  padding: 20px;
}

/* Scrollbar styling */
.timeline-scroll::-webkit-scrollbar {
  height: 8px;
}

.timeline-scroll::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.timeline-scroll::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 4px;
}

.timeline-scroll::-webkit-scrollbar-thumb:hover {
  background: #555;
}
</style>