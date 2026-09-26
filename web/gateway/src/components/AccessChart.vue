<template>
  <div ref="container" style="position: relative; height: 100%">
    <Bar ref="bar" :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  TimeScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(TimeScale, LinearScale, BarElement, Title, Tooltip, Legend)

export default {
  name: 'AccessChart',
  components: { Bar },
  props: {
    dataPoints:   { type: Array,   default: () => [] },
    granularity:  { type: String,  default: 'hour' },
    from:         { type: String,  default: '' },
    to:           { type: String,  default: '' },
    showSuccess:  { type: Boolean, default: true },
    showRejected: { type: Boolean, default: true },
    showBlocked:  { type: Boolean, default: true }
  },
  mounted() {
    this._resizeObserver = new ResizeObserver(() => {
      const chart = this.$refs.bar?.chart
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
      const datasets = []
      // Stack order: success at bottom, blocked in middle, rejected on top (most alarming)
      if (this.showSuccess) datasets.push({
        label: 'Success',
        data: this.dataPoints.map(p => p.success),
        backgroundColor: 'rgba(67, 160, 71, 0.85)',
        stack: 'requests'
      })
      if (this.showBlocked) datasets.push({
        label: 'Blocked',
        data: this.dataPoints.map(p => p.blocked),
        backgroundColor: 'rgba(251, 140, 0, 0.85)',
        stack: 'requests'
      })
      if (this.showRejected) datasets.push({
        label: 'Rejected',
        data: this.dataPoints.map(p => p.rejected),
        backgroundColor: 'rgba(229, 57, 53, 0.85)',
        stack: 'requests'
      })
      return { labels, datasets }
    },
    chartOptions() {
      const unit = this.granularity === 'week' ? 'week' : this.granularity === 'day' ? 'day' : 'hour'

      const fromD = this.from ? new Date(this.from) : null
      const toD   = this.to   ? new Date(this.to)   : null
      const multiDay = fromD && toD && (
        fromD.getFullYear() !== toD.getFullYear() ||
        fromD.getMonth()    !== toD.getMonth()    ||
        fromD.getDate()     !== toD.getDate()
      )

      const pad = n => String(n).padStart(2, '0')

      const xScale = {
        type: 'time',
        time: { unit },
        ticks: {
          maxTicksLimit: 12,
          callback(value, index, ticks) {
            const ts = ticks[index]?.value
            if (ts == null) return value
            const d = new Date(ts)
            const date = `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.`
            const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`
            if (unit === 'hour' && multiDay) return [date, time]
            if (unit === 'hour') return time
            return date
          }
        },
        stacked: true
      }
      if (this.from) xScale.min = new Date(this.from).getTime()
      if (this.to)   xScale.max = new Date(this.to + 'T23:59:59').getTime()
      return {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          x: xScale,
          y: {
            beginAtZero: true,
            stacked: true,
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
