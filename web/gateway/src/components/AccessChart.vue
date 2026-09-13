<template>
  <div ref="container" style="position: relative; height: 100%">
    <Line ref="line" :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  TimeScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(TimeScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

export default {
  name: 'AccessChart',
  components: { Line },
  props: {
    dataPoints: { type: Array, default: () => [] },
    granularity: { type: String, default: 'hour' }
  },
  mounted() {
    this._resizeObserver = new ResizeObserver(() => {
      const chart = this.$refs.line?.chart
      if (chart) chart.resize()
    })
    this._resizeObserver.observe(this.$refs.container)
  },
  beforeUnmount() {
    if (this._resizeObserver) { this._resizeObserver.disconnect(); this._resizeObserver = null }
  },
  computed: {
    chartData() {
      const labels = this.dataPoints.map(p => new Date(p.timestamp))
      const pointRadius = this.dataPoints.length > 100 ? 0 : 3
      return {
        labels,
        datasets: [
          {
            label: 'Success',
            data: this.dataPoints.map(p => p.success),
            borderColor: '#43a047',
            backgroundColor: 'rgba(67, 160, 71, 0.1)',
            fill: true,
            tension: 0.2,
            pointRadius
          },
          {
            label: 'Errors',
            data: this.dataPoints.map(p => p.errors),
            borderColor: '#e53935',
            backgroundColor: 'rgba(229, 57, 53, 0.1)',
            fill: true,
            tension: 0.2,
            pointRadius
          },
          {
            label: 'Blocked',
            data: this.dataPoints.map(p => p.blocked),
            borderColor: '#fb8c00',
            backgroundColor: 'rgba(251, 140, 0, 0.1)',
            fill: true,
            tension: 0.2,
            pointRadius
          }
        ]
      }
    },
    chartOptions() {
      return {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: {
            type: 'time',
            time: { unit: this.granularity === 'day' ? 'day' : 'hour' },
            ticks: { maxTicksLimit: 12 }
          },
          y: {
            beginAtZero: true,
            ticks: { stepSize: 1 }
          }
        },
        plugins: {
          legend: { position: 'top' },
          tooltip: { mode: 'index', intersect: false }
        }
      }
    }
  }
}
</script>
