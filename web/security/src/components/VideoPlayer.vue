<template>
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
      </div>
      <div v-else class="no-video-placeholder">
        <v-icon size="64" color="grey lighten-2">mdi-video-off</v-icon>
        <p class="text-h6 grey--text mt-4">Select a thumbnail from the timeline below</p>
      </div>
    </v-card-text>
  </v-card>
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
    }
  }
}
</script>

<style scoped>
.video-card {
  position: relative;
  overflow: hidden;
}

.video-player-wrapper {
  position: relative;
  width: 100%;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.video-container-with-controls {
  position: relative;
  width: 100%;
  aspect-ratio: 3.56;
  max-height: 60vh;
}

.video-player {
  width: 100%;
  height: 100%;
  background: #000;
  aspect-ratio: 3.56;
  object-fit: fill;
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
</style>