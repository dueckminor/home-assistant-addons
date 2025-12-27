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
                  autoplay
                  muted
                  playsinline
                  preload="metadata"
                  controlslist="nodownload nofullscreen noremoteplaybook"
                  disablepictureinpicture
                  webkit-playsinline
                  class="video-player"
                  @loadedmetadata="onVideoLoaded"
                  @error="onVideoError"
                  @ended="onVideoEnded"
                  @play="hideControlsQuickly"
                  @touchstart="hideControlsOnTouch"
                  @click="hideControlsOnTouch"
                >
                  Your browser does not support the video tag.
                </video>
              </div>
              
              <!-- Thumbnail Strip with Navigation -->
              <div v-if="selectedVideo" class="video-thumbnail-navigation mt-3">
                <div class="thumbnail-nav-container">
                  <v-btn
                    v-if="canGoPrevious"
                    icon="mdi-chevron-left"
                    variant="text"
                    size="large"
                    @click="playPreviousVideo"
                    title="Previous Video (Shift + ←)"
                    class="nav-btn-left"
                  ></v-btn>
                  
                  <div class="thumbnail-strip-container">
                    <div
                      v-for="(thumbnail, index) in getNearbyThumbnails()"
                      :key="`thumb-${thumbnail.videoName}-${index}`"
                      class="timeline-thumbnail-item"
                      :class="{ 
                        'current-time': isCurrentTimeThumbnail(thumbnail)
                      }"
                      @click="selectVideo(thumbnail)"
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
                    @click="playNextVideo"
                    title="Next Video (Shift + →)"
                    class="nav-btn-right"
                  ></v-btn>
                </div>
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
          <!-- <v-card-title>
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
          </v-card-title> -->
          
          <v-card-text>
            <div v-if="thumbnails.length === 0 && !loading" class="no-thumbnails">
              <v-alert type="info" class="ma-2">
                No recordings found for {{ formatDate(selectedDate) }}
              </v-alert>
            </div>
            
            <div v-else class="timeline-container">
              <!-- Timeline Header -->
              <div class="timeline-header">
                <div class="timeline-hours">
                  <div 
                    v-for="hour in 24" 
                    :key="hour-1" 
                    class="timeline-hour-marker"
                  >
                    {{ String(hour-1).padStart(2, '0') }}:00
                  </div>
                </div>
              </div>
              
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
                    @click.stop="selectVideo(video)"
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
                    :style="{ left: `${getCurrentTimePosition()}%` }"
                  ></div>
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
      isOrientationChanging: false,
      currentOrientation: window.orientation || 0,
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
    // Add orientation change listeners for iOS
    window.addEventListener('orientationchange', this.onOrientationChange)
    window.addEventListener('resize', this.onWindowResize)
  },
  
  beforeUnmount() {
    // Clean up event listeners
    window.removeEventListener('keydown', this.handleKeydown)
    document.removeEventListener('fullscreenchange', this.onFullscreenChange)
    document.removeEventListener('webkitfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('mozfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('msfullscreenchange', this.onFullscreenChange)
    window.removeEventListener('orientationchange', this.onOrientationChange)
    window.removeEventListener('resize', this.onWindowResize)
    if (this.fullscreenControlsTimeout) {
      clearTimeout(this.fullscreenControlsTimeout)
    }
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
        
        // Try to restore video selection from localStorage first
        let videoToSelect = null
        const persistedSelection = this.getPersistedVideoSelection()
        
        if (persistedSelection && persistedSelection.videoName) {
          // Try to find the persisted video
          videoToSelect = this.thumbnails.find(thumb => 
            thumb.videoName === persistedSelection.videoName
          )
          console.log('Restored from localStorage:', persistedSelection.videoName, videoToSelect ? 'found' : 'not found')
        }
        
        // Fallback to current video selection if localStorage doesn't have it
        if (!videoToSelect && this.selectedVideo && this.selectedVideo.name) {
          videoToSelect = this.thumbnails.find(thumb => 
            thumb.videoName === this.selectedVideo.name
          )
          console.log('Using current selection:', this.selectedVideo.name, videoToSelect ? 'found' : 'not found')
        }
        
        // Final fallback to first video only if we have no selection at all
        if (!videoToSelect && this.thumbnails.length > 0 && !this.selectedVideo) {
          videoToSelect = this.thumbnails[0]
          console.log('Auto-selecting first video:', videoToSelect.videoName)
        }
        
        if (videoToSelect) {
          this.selectVideo(videoToSelect)
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
      
      // Persist current video selection to survive iOS Safari re-renders
      const persistedSelection = {
        videoName: thumbnail.videoName,
        selectedDate: this.selectedDate,
        timestamp: Date.now()
      }
      localStorage.setItem('securityCameraSelection', JSON.stringify(persistedSelection))
      console.log('Persisted video selection:', thumbnail.videoName)
      
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
    
    getPersistedVideoSelection() {
      try {
        const stored = localStorage.getItem('securityCameraSelection')
        if (!stored) return null
        
        const selection = JSON.parse(stored)
        
        // Only use persisted selection if it's for the same date and recent (within 1 hour)
        if (selection.selectedDate === this.selectedDate && 
            selection.timestamp && 
            Date.now() - selection.timestamp < 3600000) {
          return selection
        }
        
        // Clean up old selection
        localStorage.removeItem('securityCameraSelection')
        return null
      } catch (error) {
        console.warn('Error reading persisted selection:', error)
        localStorage.removeItem('securityCameraSelection')
        return null
      }
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
      
      // Add controls back but hide them immediately
      if (this.$refs.videoPlayer) {
        const video = this.$refs.videoPlayer
        video.setAttribute('controls', '')
        
        // Hide controls immediately after adding them
        setTimeout(() => {
          this.hideControlsQuickly()
        }, 10)
        
        // Add hover/touch listeners for showing controls
        video.addEventListener('mouseenter', () => {
          video.classList.add('show-controls')
        })
        
        video.addEventListener('mouseleave', () => {
          video.classList.remove('show-controls')
          this.hideControlsQuickly()
        })
      }
    },
    
    onVideoError(event) {
      console.error('Video loading error:', event)
    },
    
    onVideoEnded() {
      console.log('Video ended, looking for next video')
      this.playNextVideo()
    },
    
    hideControlsQuickly() {
      // Force hide controls overlay quickly on iOS and other browsers
      if (this.$refs.videoPlayer) {
        const video = this.$refs.videoPlayer
        
        // Remove controls attribute completely
        video.removeAttribute('controls')
        video.classList.remove('show-controls')
        
        // Remove focus and blur
        video.blur()
        
        // Multiple approaches for iOS
        setTimeout(() => {
          if (video && !video.paused) {
            // Small seek to reset state
            const currentTime = video.currentTime
            video.currentTime = currentTime + 0.001
            
            // Force a style recalculation
            video.style.display = 'none'
            video.offsetHeight // Force reflow
            video.style.display = 'block'
          }
        }, 50)
      }
    },
    
    hideControlsOnTouch() {
      this.hideControlsQuickly()
    },
    
    onOrientationChange() {
      // iOS Safari fires this when device is rotated
      console.log('Orientation change detected')
      this.isOrientationChanging = true
      this.currentOrientation = window.orientation || 0
      
      // Clear the flag after a delay to allow for orientation transition
      setTimeout(() => {
        this.isOrientationChanging = false
        console.log('Orientation change complete')
      }, 1000)
    },
    
    onWindowResize() {
      // Additional check for iOS Safari resize events during orientation change
      const newOrientation = window.orientation || 0
      if (newOrientation !== this.currentOrientation) {
        this.onOrientationChange()
      }
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
    },
    
    getVideoBlockStyle(video) {
      const dayStart = new Date(this.selectedDate)
      const dayEnd = new Date(dayStart)
      dayEnd.setDate(dayEnd.getDate() + 1)
      
      const videoStart = video.timestamp
      const minutesFromStart = (videoStart - dayStart) / (1000 * 60)
      const minutesInDay = 24 * 60
      
      const leftPercent = (minutesFromStart / minutesInDay) * 100
      
      // Assume 2 minute video duration for block width (can be adjusted)
      const videoDurationMinutes = 2
      const widthPercent = (videoDurationMinutes / minutesInDay) * 100
      
      return {
        left: `${Math.max(0, leftPercent)}%`,
        width: `${Math.max(0.2, widthPercent)}%`
      }
    },
    
    getCurrentTimePosition() {
      if (!this.selectedVideo) return 0
      
      const dayStart = new Date(this.selectedDate)
      const videoStart = this.selectedVideo.timestamp
      const minutesFromStart = (videoStart - dayStart) / (1000 * 60)
      const minutesInDay = 24 * 60
      
      return (minutesFromStart / minutesInDay) * 100
    },
    
    onTimelineClick(event) {
      const rect = event.currentTarget.getBoundingClientRect()
      const clickX = event.clientX - rect.left
      const timelineWidth = rect.width
      const clickPercent = clickX / timelineWidth
      
      const dayStart = new Date(this.selectedDate)
      const clickTime = new Date(dayStart.getTime() + (clickPercent * 24 * 60 * 60 * 1000))
      
      // Find closest video to clicked time
      let closestVideo = null
      let minTimeDiff = Infinity
      
      this.thumbnails.forEach(video => {
        const timeDiff = Math.abs(video.timestamp - clickTime)
        if (timeDiff < minTimeDiff) {
          minTimeDiff = timeDiff
          closestVideo = video
        }
      })
      
      if (closestVideo && minTimeDiff < 30 * 60 * 1000) { // Within 30 minutes
        this.selectVideo(closestVideo)
      }
    },
    
    getTotalDuration() {
      // Estimate total duration (2 minutes per video)
      return this.thumbnails.length * 2 * 60 * 1000
    },
    
    formatDuration(durationMs) {
      const minutes = Math.floor(durationMs / (1000 * 60))
      const hours = Math.floor(minutes / 60)
      const remainingMinutes = minutes % 60
      
      if (hours > 0) {
        return `${hours}h ${remainingMinutes}m`
      } else {
        return `${minutes}m`
      }
    },
    
    getNearbyThumbnails() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return []
      
      // Find the index of the currently selected video
      const currentIndex = this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
      
      if (currentIndex === -1) return this.thumbnails.slice(0, Math.min(7, this.thumbnails.length))
      
      const maxThumbnails = 7
      const halfWindow = 3 // Fixed: 3 on each side
      
      // Simple approach: center around current, then slice to max length
      const startIndex = Math.max(0, currentIndex - halfWindow)
      const endIndex = startIndex + maxThumbnails
      
      return this.thumbnails.slice(startIndex, Math.min(endIndex, this.thumbnails.length))
    },
    
    isCurrentTimeThumbnail(thumbnail) {
      if (!this.selectedVideo || !thumbnail) return false
      return thumbnail.videoName === this.selectedVideo.name
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
  /* Maintain camera's native aspect ratio: 7680 × 2160 = 3.56:1 */
  aspect-ratio: 3.56;
  max-height: 60vh;
}

.video-player {
  width: 100%;
  height: 100%;
  background: #000;
  object-fit: fill; /* Use fill since we're matching the native aspect ratio */
}

/* Minimize video controls overlay behavior */
.video-player::-webkit-media-controls-overlay-enclosure {
  display: none !important;
}

.video-player::-webkit-media-controls {
  opacity: 0 !important;
  transition: opacity 0.05s ease !important;
}

.video-player:hover::-webkit-media-controls,
.video-player:focus::-webkit-media-controls {
  opacity: 1 !important;
}

/* iOS Safari specific controls hiding */
.video-player::-webkit-media-controls-start-playback-button {
  display: none !important;
}

.video-player::-webkit-media-controls-overlay-cast-button {
  display: none !important;
}

.video-player::-webkit-media-controls-fullscreen-button {
  display: none !important;
}

.video-player::-webkit-media-controls-overlay-play-button {
  display: none !important;
}

/* Force hide all overlays */
.video-player {
  -webkit-media-controls-overlay-cast-button: none;
  -webkit-appearance: none;
}

/* iOS specific - hide all overlay elements */
@media screen and (-webkit-min-device-pixel-ratio: 2) {
  .video-player::-webkit-media-controls {
    display: none !important;
  }
  
  .video-player.show-controls::-webkit-media-controls {
    display: block !important;
    opacity: 1 !important;
  }
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
  max-height: 350px;
}

.timeline-container {
  padding: 16px 0;
}

.timeline-header {
  margin-bottom: 8px;
}

.timeline-hours {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: #666;
  margin-bottom: 4px;
}

.timeline-hour-marker {
  flex: 1;
  text-align: center;
  font-weight: 500;
}

.timeline-bar-container {
  position: relative;
  cursor: pointer;
  margin: 8px 0;
}

.timeline-bar {
  position: relative;
  height: 40px;
  background: linear-gradient(to bottom, #f5f5f5, #e8e8e8);
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: hidden;
}

.timeline-grid-line {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: #ccc;
  opacity: 0.5;
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

.timeline-current-indicator {
  position: absolute;
  top: -2px;
  bottom: -2px;
  width: 2px;
  background: #ff5722;
  border-radius: 1px;
  box-shadow: 0 0 4px rgba(255,87,34,0.6);
  z-index: 15;
}

.timeline-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.timeline-stats {
  display: flex;
  align-items: center;
}

/* Mobile landscape optimization */
@media (max-height: 500px) and (orientation: landscape) {
  /* Hide app bar on small landscape screens */
  .v-app-bar {
    display: none !important;
  }
  
  /* Hide timeline card on small landscape screens */
  .timeline-card {
    display: none !important;
  }
  
  /* Adjust main content to use full screen */
  .v-main {
    padding-top: 0 !important;
  }
  
  /* Make video container take full available height */
  .video-card {
    margin: 0 !important;
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  
  /* Optimize video player for full screen landscape */
  .video-container-with-controls {
    flex: 1;
    max-height: none;
    height: 100%;
  }
  
  /* Ensure video fills available space */
  .video-player {
    height: 100%;
    object-fit: contain; /* Switch back to contain for full screen */
  }
  
  /* Make overlay controls more prominent in landscape */
  .video-nav-btn {
    width: 50px;
    height: 50px;
    background: rgba(0, 0, 0, 0.8);
  }
  
  .video-nav-btn .v-icon {
    font-size: 28px;
  }
}

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

/* Thumbnail Strip Styles */
.timeline-thumbnail-strip {
  margin: 16px 0;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.thumbnail-strip-container {
  display: flex;
  gap: 8px;
  justify-content: center;
  align-items: center;
  overflow-x: auto;
  padding: 4px 0;
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
  border: 2px solid #ff5722;
  box-shadow: 0 4px 12px rgba(255,87,34,0.4);
  transform: scale(1.1);
}

.timeline-thumbnail-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.timeline-thumbnail-image {
  width: 60px;
  height: 45px;
  object-fit: cover;
  border-radius: 4px;
  background: #f0f0f0;
}

.timeline-thumbnail-placeholder {
  width: 60px;
  height: 45px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f5f5, #e0e0e0);
  border: 1px dashed #ccc;
  border-radius: 4px;
  min-height: 45px;
}

.placeholder-video-icon {
  font-size: 20px;
  line-height: 1;
  color: #999;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.timeline-thumbnail-time {
  font-size: 10px;
  font-weight: 500;
  color: #666;
  margin-top: 4px;
  text-align: center;
  min-width: 60px;
}

.timeline-thumbnail-item.current-time .timeline-thumbnail-time {
  color: #ff5722;
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