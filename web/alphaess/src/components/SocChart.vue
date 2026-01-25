<template>
  <div style="position: relative; height: 200px;">
    <Line :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler
)

export default {
  name: 'SocChart',
  components: {
    Line
  },
  props: {
    measurements: {
      type: Array,
      required: true
    },
    selectedDate: {
      type: Date,
      required: true
    }
  },
  computed: {
    dayStart() {
      const date = new Date(this.selectedDate)
      date.setHours(0, 0, 0, 0)
      return date
    },
    dayEnd() {
      const date = new Date(this.selectedDate)
      date.setHours(23, 0, 0, 0)
      return date
    },
    timezone() {
      // Get timezone abbreviation (e.g., CET, CEST) for the selected date
      const date = new Date(this.selectedDate)
      date.setHours(12, 0, 0, 0) // Use noon to get the correct timezone
      
      // Try to get the timezone abbreviation using toLocaleTimeString
      const timeString = date.toLocaleTimeString('en-US', {
        timeZoneName: 'short',
        hour: 'numeric'
      })
      
      // Extract timezone abbreviation from the time string
      const match = timeString.match(/([A-Z]{3,5})$/)
      if (match && match[1] && !match[1].startsWith('GMT')) {
        return match[1]
      }
      
      // Fallback to GMT offset
      const offset = -date.getTimezoneOffset() / 60
      return `UTC${offset >= 0 ? '+' : ''}${offset}`
    },
    chartData() {
      const datasets = []
      const gapFilledTimes = new Set()
      
      // Collect gap times
      this.measurements.forEach(agg => {
        if (agg.gap) {
          const time = new Date(agg.start_time).getTime()
          gapFilledTimes.add(time)
        }
      })

      // Extract SOC data
      const socData = this.measurements
        .filter(agg => agg.battery_soc !== undefined)
        .map(agg => {
          const time = new Date(agg.start_time)
          const isGap = gapFilledTimes.has(time.getTime())
          
          return {
            x: time,
            y: agg.battery_soc,
            isGap: isGap
          }
        })
      
      if (socData.length > 0) {
        datasets.push({
          label: 'Battery SOC',
          data: socData,
          borderColor: 'rgba(139, 195, 74, 0.8)',
          backgroundColor: 'rgba(139, 195, 74, 0.2)',
          borderWidth: 2,
          tension: 0.3,
          fill: true,
          pointRadius: 0,
          segment: {
            borderDash: ctx => {
              // Use dashed line for gaps
              const curr = ctx.p1DataIndex
              if (curr >= 0 && curr < socData.length) {
                return socData[curr].isGap ? [5, 5] : []
              }
              return []
            }
          }
        })
      }

      return { datasets }
    },
    chartOptions() {
      return {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
          duration: 0
        },
        layout: {
          padding: {
            left: 0,
            right: 0
          }
        },
        interaction: {
          mode: 'index',
          intersect: false
        },
        plugins: {
          legend: {
            display: true,
            position: 'bottom'
          },
          tooltip: {
            callbacks: {
              label: (context) => {
                const label = context.dataset.label || ''
                const value = context.parsed.y.toFixed(1)
                return `${label}: ${value}%`
              }
            }
          }
        },
        scales: {
          x: {
            type: 'time',
            time: {
              unit: 'hour',
              displayFormats: {
                hour: 'HH:mm'
              }
            },
            min: this.dayStart,
            max: this.dayEnd,
            title: {
              display: true,
              text: `Time (${this.timezone})`
            },
            grid: {
              display: true
            }
          },
          y: {
            type: 'linear',
            min: 0,
            max: 100,
            title: {
              display: true,
              text: 'SOC (%)'
            },
            ticks: {
              callback: function(value) {
                return value.toFixed(0)
              }
            }
          }
        }
      }
    }
  }
}
</script>
