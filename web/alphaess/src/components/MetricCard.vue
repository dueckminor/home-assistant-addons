<template>
  <v-card class="metric-card" :class="color">
    <v-card-text class="d-flex align-center">
      <div class="flex-grow-1">
        <div class="text-h4 font-weight-bold">{{ value }}</div>
        <div class="text-caption text-medium-emphasis">{{ unit }}</div>
        <div class="text-body-2 font-weight-medium">{{ title }}</div>
      </div>
      <div class="metric-icon">
        <v-icon :color="color" size="48">{{ icon }}</v-icon>
      </div>
    </v-card-text>
  </v-card>
</template>

<script>
export default {
  name: 'MetricCard',
  props: {
    title: {
      type: String,
      required: true
    },
    value: {
      type: [String, Number],
      required: true
    },
    unit: {
      type: String,
      default: ''
    },
    icon: {
      type: String,
      required: true
    },
    color: {
      type: String,
      default: 'primary'
    }
  }
}
</script>

<style scoped>
.metric-card {
  height: 120px;
  border-left: 4px solid;
}

.metric-card.orange {
  border-left-color: #ff9800;
}

.metric-card.green {
  border-left-color: #4caf50;
}

.metric-card.blue {
  border-left-color: #2196f3;
}

.metric-card.purple {
  border-left-color: #9c27b0;
}

.metric-icon {
  opacity: 0.7;
}
</style>