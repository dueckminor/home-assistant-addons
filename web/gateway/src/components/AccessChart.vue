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
      return {
        labels,
        datasets: [
          {
            label: 'Requests',
            data: this.dataPoints.map(p => p.count),
            borderColor: '#1976d2',
            backgroundColor: 'rgba(25, 118, 210, 0.1)',
            fill: true,
            tension: 0.2,
            pointRadius: this.dataPoints.length > 100 ? 0 : 3
          },
          {
            label: 'Errors',
            data: this.dataPoints.map(p => p.errors),
            borderColor: '#d32f2f',
            backgroundColor: 'rgba(211, 47, 47, 0.1)',
            fill: true,
            tension: 0.2,
            pointRadius: this.dataPoints.length > 100 ? 0 : 3
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
