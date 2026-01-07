<template>
  <v-app>
    <v-main class="app-main">
      <!-- Video Player Section - Takes remaining space -->
      <div class="video-section">
        <VideoPlayer 
          :selected-video="selectedVideo"
          @video-ended="playNextVideo"
          @video-loaded="onVideoLoaded"
          @video-error="onVideoError"
        />
        
        <!-- Loading indicator -->
        <v-progress-linear
          v-if="loading"
          indeterminate
          color="primary"
          class="loading-indicator"
        ></v-progress-linear>
      </div>
      
      <!-- Bottom Controls Section -->
      <div class="bottom-controls">
        <!-- Navigation Controls -->
        <NavigationControls
          :selected-date="selectedDate"
          :selected-video="selectedVideo"
          :can-go-previous="canGoPrevious"
          :can-go-next="canGoNext"
          @previous-day="goToPreviousDay"
          @next-day="goToNextDay"
          @previous-video="playPreviousVideo"
          @next-video="playNextVideo"
          @date-selected="onDateSelected"
        />
        
        <!-- Timeline Component -->
        <Timeline
          :selected-video="selectedVideo"
          :thumbnails="thumbnails"
          :selected-date="selectedDate"
          :loading="loading"
          :can-go-previous="canGoPrevious"
          :can-go-next="canGoNext"
          @select-video="selectVideo"
          @timeline-click="onTimelineClick"
          @previous-video="playPreviousVideo"
          @next-video="playNextVideo"
          @previous-day="goToPreviousDay"
          @next-day="goToNextDay"
        />
      </div>
    </v-main>
  </v-app>
</template>

<script>
import VideoPlayer from './components/VideoPlayer.vue'
import ThumbnailNavigation from './components/ThumbnailNavigation.vue'
import Timeline from './components/Timeline.vue'
import NavigationControls from './components/NavigationControls.vue'
import { getBaseUrl } from '../../shared/utils/homeassistant.js'

export default {
  name: 'SecurityApp',
  components: {
    VideoPlayer,
    ThumbnailNavigation,
    Timeline,
    NavigationControls
  },
  data() {
    return {
      selectedDate: new Date().toISOString().substr(0, 10),
      selectedVideo: null,
      thumbnails: [],
      loading: false,
      baseUrl: getBaseUrl(),
      isOrientationChanging: false,
      currentOrientation: window.orientation || 0,
    }
  },
  
  mounted() {
    this.loadDayFiles()
    // Add keyboard event listeners
    window.addEventListener('keydown', this.handleKeydown)
    // Add orientation change listeners for iOS
    window.addEventListener('orientationchange', this.onOrientationChange)
    window.addEventListener('resize', this.onWindowResize)
  },
  
  beforeUnmount() {
    // Clean up event listeners
    window.removeEventListener('keydown', this.handleKeydown)
    window.removeEventListener('orientationchange', this.onOrientationChange)
    window.removeEventListener('resize', this.onWindowResize)
  },
  
  computed: {
    isToday() {
      const today = new Date().toISOString().substr(0, 10)
      return this.selectedDate === today
    },
    
    canGoPrevious() {
      return this.currentVideoIndex > 0
    },
    
    canGoNext() {
      return this.currentVideoIndex < this.thumbnails.length - 1 && this.currentVideoIndex >= 0
    },
    
    currentVideoIndex() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return -1
      return this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
    }
  },
  
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
        
        const response = await fetch(`${this.baseUrl}/api/ftp/${encodeURIComponent(path)}`)
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`)
        }
        const data = await response.json()
        
        // Filter for JPG and MP4 files
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
    
    extractTimestampFromFilename(filename) {
      // Extract timestamp from filename format like: cam1_20231225143022.mp4
      // or other formats like: 20231225143022_cam1.jpg
      const matches = filename.match(/(\d{14})/)
      if (matches) {
        const timestampStr = matches[1]
        const year = parseInt(timestampStr.substr(0, 4))
        const month = parseInt(timestampStr.substr(4, 2)) - 1 // Month is 0-indexed
        const day = parseInt(timestampStr.substr(6, 2))
        const hour = parseInt(timestampStr.substr(8, 2))
        const minute = parseInt(timestampStr.substr(10, 2))
        const second = parseInt(timestampStr.substr(12, 2))
        
        return new Date(year, month, day, hour, minute, second).getTime()
      }
      
      // No fallback - the time MUST be in the filename
      console.warn('Could not extract timestamp from filename:', filename)
      return null
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
        console.log('Already at the last video of the day')
      }
    },
    
    playPreviousVideo() {
      if (!this.selectedVideo || this.thumbnails.length === 0) return
      
      // Find current video index in thumbnails
      const currentIndex = this.thumbnails.findIndex(thumb => 
        thumb.videoName === this.selectedVideo.name
      )
      
      if (currentIndex === -1) return
      
      // Get previous video (or loop to last if at the beginning)
      const prevIndex = currentIndex - 1
      if (prevIndex >= 0) {
        // Play previous video
        this.selectVideo(this.thumbnails[prevIndex])
      } else {
        console.log('Already at the first video of the day')
      }
    },
    
    onVideoLoaded() {
      // Handle video loaded event from VideoPlayer component
    },
    
    onVideoError(event) {
      // Handle video error event from VideoPlayer component
      console.error('Video error from component:', event)
    },
    
    onTimelineClick(event) {
      // Handle timeline click event from Timeline component
      console.log('Timeline clicked:', event)
    },
    
    goToPreviousDay() {
      const currentDate = new Date(this.selectedDate)
      currentDate.setDate(currentDate.getDate() - 1)
      this.selectedDate = currentDate.toISOString().substr(0, 10)
      this.loadDayFiles()
    },
    
    goToNextDay() {
      const currentDate = new Date(this.selectedDate)
      currentDate.setDate(currentDate.getDate() + 1)
      this.selectedDate = currentDate.toISOString().substr(0, 10)
      this.loadDayFiles()
    },
    
    onDateSelected(date) {
      this.selectedDate = date
      this.loadDayFiles()
    }
  }
}
</script>

<style scoped>
/* Main app layout - column with video at top, controls at bottom */
.app-main {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh; /* Use dynamic viewport height when available */
  min-height: 100vh;
  min-height: 100dvh;
}

/* iOS Safari specific height fix */
@supports (-webkit-touch-callout: none) {
  .app-main {
    height: -webkit-fill-available;
    min-height: -webkit-fill-available;
  }
}

.video-section {
  flex: 1; /* Take remaining space */
  display: flex;
  flex-direction: column;
  min-height: 0; /* Allow shrinking */
  position: relative;
  overflow: hidden; /* Prevent scroll within video section */
}

/* Mobile-specific adjustments */
@media (max-width: 768px) {
  .video-section {
    flex: 1; /* Take available space */
    min-height: 0; /* Allow shrinking */
  }
  
  @supports (-webkit-touch-callout: none) {
    .video-section {
      max-height: calc(-webkit-fill-available - 35vh);
    }
  }
}

.bottom-controls {
  flex-shrink: 0; /* Don't shrink */
  background: rgba(255, 255, 255, 0.95);
  border-top: 1px solid #e0e0e0;
  padding: 8px;
  overflow-y: auto; /* Allow scrolling if needed */
  max-height: 40vh; /* Use viewport height instead of fixed pixels */
}

/* Compact bottom controls on small screens */
@media (max-width: 768px) {
  .bottom-controls {
    padding: 4px 8px;
    max-height: 35vh; /* Allow more space but still limit */
  }
}

@media (orientation: landscape) and (max-height: 500px) {
  .bottom-controls {
    padding: 2px 4px;
    max-height: 25vh; /* Use viewport height for consistency */
  }
}

.loading-indicator {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
}

/* Responsive design for different screen sizes */

/* Mobile and small screens: Stack controls vertically */
@media (max-width: 768px) {
  .bottom-controls {
    padding: 4px;
  }
  
  .navigation-container {
    margin-bottom: 8px;
  }
  
  .timeline-container-wrapper {
    margin-top: 8px;
  }
}

/* Ensure timeline is always fully visible */
@media (max-width: 768px) and (orientation: portrait) {
  .bottom-controls {
    display: flex;
    flex-direction: column;
    max-height: none; /* Remove height restriction on portrait mobile */
    overflow-y: visible;
  }
  
  .timeline-container-wrapper {
    flex: 1;
    min-height: 0;
  }
}

/* Landscape mode optimizations */
@media (orientation: landscape) {
  /* Make bottom controls more compact in landscape */
  .bottom-controls {
    padding: 4px 8px;
    display: flex;
    gap: 8px;
    align-items: flex-start;
  }
  
  /* Navigation takes fixed width, timeline takes remaining space */
  .navigation-container {
    flex: 0 0 280px;
    margin: 0;
  }
  
  .timeline-container-wrapper {
    flex: 1;
    margin: 0;
    min-width: 0; /* Allow shrinking */
  }
  
  /* Make controls more compact */
  .navigation-container {
    padding: 8px;
  }
  
  .timeline-container-wrapper {
    padding: 8px;
  }
  
  :deep(.nav-row) {
    margin-bottom: 4px !important;
  }
  
  :deep(.nav-display) {
    padding: 4px 8px !important;
    min-height: 32px !important;
    font-size: 12px !important;
  }
}

/* Small landscape screens - extra compact layout */
@media (orientation: landscape) and (max-height: 480px) {
  .bottom-controls {
    padding: 2px 4px;
  }
  
  .navigation-container {
    flex: 0 0 260px;
    padding: 4px;
  }
  
  .timeline-container-wrapper {
    padding: 4px;
  }
  
  :deep(.nav-display) {
    font-size: 11px !important;
    min-height: 28px !important;
  }
}
</style>