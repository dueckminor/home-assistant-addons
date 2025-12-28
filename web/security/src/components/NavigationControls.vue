<template>
  <div class="navigation-container">
    <!-- Date Navigation Row -->
    <div class="nav-row date-nav">
      <v-btn
        icon="mdi-chevron-double-left"
        variant="outlined"
        size="small"
        @click="$emit('previous-day')"
        title="Previous Day"
        class="nav-btn"
      ></v-btn>
      
<div class="nav-display date-display" @click="showDatePicker = true" style="cursor: pointer;">
        <v-icon class="mr-2">mdi-calendar</v-icon>
        {{ formatDate(selectedDate) }}
      </div>
      
      <v-btn
        icon="mdi-chevron-double-right"
        variant="outlined"
        size="small"
        @click="$emit('next-day')"
        title="Next Day"
        class="nav-btn"
      ></v-btn>
    </div>
    
    <!-- Video Navigation Row -->
    <div class="nav-row video-nav">
      <v-btn
        icon="mdi-chevron-left"
        variant="outlined"
        size="small"
        :disabled="!canGoPrevious"
        @click="$emit('previous-video')"
        title="Previous Video"
        class="nav-btn"
      ></v-btn>
      
      <div class="nav-display time-display">
        <v-icon class="mr-2">mdi-clock</v-icon>
        <span v-if="selectedVideo">
          {{ formatTime(selectedVideo.timestamp) }}
        </span>
        <span v-else class="text-disabled">
          No video selected
        </span>
      </div>
        
      <v-btn
        icon="mdi-chevron-right"
        variant="outlined"
        size="small"
        :disabled="!canGoNext"
        @click="$emit('next-video')"
        title="Next Video"
        class="nav-btn"
      ></v-btn>
    </div>
  </div>
  
  <!-- Date Picker Dialog -->
  <v-dialog v-model="showDatePicker" max-width="400px">
    <v-card>
      <v-card-title>Select Date</v-card-title>
      <v-card-text>
        <v-date-picker
          v-model="internalDate"
          @update:model-value="onDateSelected"
          full-width
        ></v-date-picker>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn text @click="showDatePicker = false">Cancel</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
export default {
  name: 'NavigationControls',
  props: {
    selectedDate: {
      type: String,
      required: true
    },
    selectedVideo: {
      type: Object,
      default: null
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
  emits: ['previous-day', 'next-day', 'previous-video', 'next-video', 'date-selected'],
  data() {
    return {
      showDatePicker: false,
      internalDate: null
    }
  },
  watch: {
    selectedDate: {
      immediate: true,
      handler(newDate) {
        this.internalDate = newDate
      }
    }
  },
  methods: {
    formatDate(dateString) {
      const date = new Date(dateString)
      return date.toLocaleDateString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      })
    },
    
    formatTime(timestamp) {
      const date = new Date(timestamp)
      return date.toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    },
    
    onDateSelected(date) {
      this.$emit('date-selected', date)
      this.showDatePicker = false
    }
  }
}
</script>

<style scoped>
.navigation-card {
  max-width: 400px;
  margin: 0 auto;
}

.nav-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.nav-row:last-child {
  margin-bottom: 0;
}

.nav-display {
  flex: 1;
  text-align: center;
  font-weight: 500;
  padding: 8px 16px;
  background: #f5f5f5;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
}

.date-display {
  background: #e3f2fd;
  color: #1565c0;
  font-size: 14px;
}

.time-display {
  background: #f3e5f5;
  color: #7b1fa2;
  font-size: 13px;
  font-family: monospace;
}

.nav-btn {
  flex-shrink: 0;
  min-width: 40px;
}

.text-disabled {
  color: #9e9e9e;
  font-style: italic;
}

/* Responsive adjustments */
@media (max-width: 480px) {
  .nav-display {
    font-size: 12px;
    padding: 6px 12px;
  }
  
  .date-display {
    font-size: 13px;
  }
  
  .time-display {
    font-size: 12px;
  }
}
</style>