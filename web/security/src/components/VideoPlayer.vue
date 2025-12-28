<template>
  <div class="video-player-container">
    <div v-if="selectedVideo" class="video-player-wrapper">
      <div 
        ref="fullscreenContainer"
        class="video-container-with-controls"
        :class="{ 'fullscreen-active': isFullscreen }"
      >
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
          @touchstart="handleVideoTouch"
          @click="handleVideoClick"
        >
          Your browser does not support the video tag.
        </video>
        
        <!-- Fullscreen exit hint -->
        <div v-if="isFullscreen" class="fullscreen-hint">
          <span>Press ESC or click to exit fullscreen</span>
        </div>
      </div>
    </div>
    <div v-else class="no-video-placeholder">
      <v-icon size="64" color="grey lighten-2">mdi-video-off</v-icon>
      <p class="text-h6 grey--text mt-4">Select a thumbnail from the timeline below</p>
    </div>
  </div>
</template>

<script>
export default {
  name: 'VideoPlayer',
  props: {
    selectedVideo: {
      type: Object,
      default: null
    }
  },
  emits: ['video-ended', 'video-loaded', 'video-error'],
  data() {
    return {
      isFullscreen: false
    }
  },
  mounted() {
    // Listen for fullscreen change events
    document.addEventListener('fullscreenchange', this.onFullscreenChange)
    document.addEventListener('webkitfullscreenchange', this.onFullscreenChange)
    document.addEventListener('mozfullscreenchange', this.onFullscreenChange)
    document.addEventListener('MSFullscreenChange', this.onFullscreenChange)
    
    // iOS-specific video fullscreen events
    if (this.$refs.videoPlayer) {
      this.$refs.videoPlayer.addEventListener('webkitbeginfullscreen', this.onVideoFullscreenEnter)
      this.$refs.videoPlayer.addEventListener('webkitendfullscreen', this.onVideoFullscreenExit)
    }
  },
  beforeUnmount() {
    // Clean up event listeners
    document.removeEventListener('fullscreenchange', this.onFullscreenChange)
    document.removeEventListener('webkitfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('mozfullscreenchange', this.onFullscreenChange)
    document.removeEventListener('MSFullscreenChange', this.onFullscreenChange)
    
    // Clean up iOS-specific events
    if (this.$refs.videoPlayer) {
      this.$refs.videoPlayer.removeEventListener('webkitbeginfullscreen', this.onVideoFullscreenEnter)
      this.$refs.videoPlayer.removeEventListener('webkitendfullscreen', this.onVideoFullscreenExit)
    }
  },
  methods: {
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
      
      this.$emit('video-loaded')
    },
    
    onVideoError(event) {
      console.error('Video loading error:', event)
      this.$emit('video-error', event)
    },
    
    onVideoEnded() {
      console.log('Video ended, looking for next video')
      this.$emit('video-ended')
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
    
    playVideo() {
      if (this.$refs.videoPlayer) {
        const playPromise = this.$refs.videoPlayer.play()
        
        if (playPromise !== undefined) {
          playPromise
            .then(() => {
              console.log('Autoplay started successfully')
            })
            .catch(error => {
              console.log('Autoplay prevented:', error.message)
            })
        }
      }
    },
    
    handleVideoClick(event) {
      // Toggle fullscreen on click
      this.toggleFullscreen()
      // Also hide controls
      this.hideControlsOnTouch(event)
    },
    
    handleVideoTouch(event) {
      // On touch, just hide controls (don't toggle fullscreen to avoid accidental fullscreen)
      this.hideControlsOnTouch(event)
    },
    
    isIOS() {
      return /iPad|iPhone|iPod/.test(navigator.userAgent)
    },
    
    toggleFullscreen() {
      if (this.isFullscreen) {
        this.exitFullscreen()
      } else {
        this.enterFullscreen()
      }
    },
    
    enterFullscreen() {
      // For iOS, use video-specific fullscreen
      if (this.isIOS() && this.$refs.videoPlayer) {
        const video = this.$refs.videoPlayer
        if (video.webkitSupportsFullscreen) {
          video.webkitEnterFullscreen()
          return
        }
      }
      
      // For other browsers, use container fullscreen
      const element = this.$refs.fullscreenContainer
      if (element) {
        if (element.requestFullscreen) {
          element.requestFullscreen()
        } else if (element.webkitRequestFullscreen) {
          element.webkitRequestFullscreen()
        } else if (element.mozRequestFullScreen) {
          element.mozRequestFullScreen()
        } else if (element.msRequestFullscreen) {
          element.msRequestFullscreen()
        }
      }
    },
    
    exitFullscreen() {
      // For iOS video fullscreen, it exits automatically or via controls
      if (this.isIOS() && this.$refs.videoPlayer) {
        const video = this.$refs.videoPlayer
        if (video.webkitExitFullscreen) {
          video.webkitExitFullscreen()
          return
        }
      }
      
      // For other browsers
      if (document.exitFullscreen) {
        document.exitFullscreen()
      } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen()
      } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen()
      } else if (document.msExitFullscreen) {
        document.msExitFullscreen()
      }
    },
    
    onFullscreenChange() {
      // Check if we're in fullscreen mode
      const fullscreenElement = document.fullscreenElement || 
                                document.webkitFullscreenElement || 
                                document.mozFullScreenElement || 
                                document.msFullscreenElement
      
      this.isFullscreen = !!fullscreenElement
    },
    
    onVideoFullscreenEnter() {
      // iOS video entered fullscreen
      this.isFullscreen = true
    },
    
    onVideoFullscreenExit() {
      // iOS video exited fullscreen
      this.isFullscreen = false
    }
  }
}
</script>

<style scoped>
.video-player-container {
  position: relative;
  overflow: hidden;
  width: 100%;
  height: 100%;
}

.video-player-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-container-with-controls {
  position: relative;
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Mobile landscape optimization - make video smaller */
@media (max-height: 500px) and (orientation: landscape) {
  .video-container-with-controls {
    max-height: 70vh; /* Use viewport height instead of fixed */
  }
  
  .video-player {
    max-height: 70vh;
  }
}

/* iOS specific mobile adjustments */
@media (max-width: 768px) {
  .video-container-with-controls {
    height: 100%;
  }
  
  .video-player {
    object-fit: contain; /* Ensure video fits properly */
  }
}

.video-player {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  background: #000;
  object-fit: contain; /* Maintain aspect ratio */
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

.no-video-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  text-align: center;
}

/* Landscape mode - reduce placeholder height to save space */
@media (orientation: landscape) and (max-height: 480px) {
  .no-video-placeholder {
    min-height: 120px;
  }
}

/* Fullscreen styles */
.fullscreen-active {
  background: #000 !important;
}

.fullscreen-active .video-player {
  width: 100% !important;
  height: 100% !important;
  max-height: none !important;
  object-fit: contain;
}

.fullscreen-hint {
  position: absolute;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.7);
  color: white;
  padding: 8px 16px;
  border-radius: 4px;
  font-size: 14px;
  z-index: 100;
  pointer-events: none;
  animation: fadeInOut 3s ease-in-out;
}

@keyframes fadeInOut {
  0% { opacity: 0; }
  20% { opacity: 1; }
  80% { opacity: 1; }
  100% { opacity: 0; }
}

/* Fullscreen cursor */
.video-player {
  cursor: pointer;
}

.fullscreen-active .video-player {
  cursor: none;
}

.fullscreen-active .video-player:hover {
  cursor: pointer;
}
</style>