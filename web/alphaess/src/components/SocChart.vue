<template>
  <div style="position: relative; height: 132px;">
    <Bar :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
  TimeScale
} from 'chart.js'
import 'chartjs-adapter-date-fns'

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
  TimeScale
)

export default {
  name: 'SocChart',
  components: {
    Bar
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

      // Create three stacked datasets for min, mean-min, and max-mean
      const minData = []
      const meanMinusMinData = []
      const maxMinusMeanData = []
      
      this.measurements
        .filter(agg => agg.battery_soc !== undefined)
        .forEach(agg => {
          const time = new Date(agg.start_time)
          const min = agg.battery_soc_min || agg.battery_soc
          const mean = agg.battery_soc
          const max = agg.battery_soc_max || agg.battery_soc
          
          minData.push({ x: time, y: min })
          meanMinusMinData.push({ x: time, y: mean - min })
          maxMinusMeanData.push({ x: time, y: max - mean })
        })
      
      if (minData.length > 0) {
        // Bottom segment: 0 to min (darkest green)
        datasets.push({
          label: 'Min SOC',
          data: minData,
          backgroundColor: this.measurements.map(agg => 
            agg.gap ? 'rgba(139, 195, 74, 0.5)' : 'rgba(139, 195, 74, 0.85)'
          ),
          borderWidth: 0,
          barPercentage: 0.95,
          categoryPercentage: 1.0,
          stack: 'soc'
        })
        
        // Middle segment: min to mean (medium green)
        datasets.push({
          label: 'Mean SOC',
          data: meanMinusMinData,
          backgroundColor: this.measurements.map(agg => 
            agg.gap ? 'rgba(139, 195, 74, 0.4)' : 'rgba(139, 195, 74, 0.65)'
          ),
          borderWidth: 0,
          barPercentage: 0.95,
          categoryPercentage: 1.0,
          stack: 'soc'
        })
        
        // Top segment: mean to max (lightest green)
        datasets.push({
          label: 'Max SOC',
          data: maxMinusMeanData,
          backgroundColor: this.measurements.map(agg => 
            agg.gap ? 'rgba(139, 195, 74, 0.2)' : 'rgba(139, 195, 74, 0.4)'
          ),
          borderWidth: 0,
          barPercentage: 0.95,
          categoryPercentage: 1.0,
          stack: 'soc'
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
            display: false
          },
          tooltip: {
            callbacks: {
              footer: (tooltipItems) => {
                if (tooltipItems.length === 0) return ''
                
                const index = tooltipItems[0].dataIndex
                const data = this.measurements[index]
                
                if (!data) return ''
                
                const mean = data.battery_soc?.toFixed(1) || 'N/A'
                const min = data.battery_soc_min?.toFixed(1) || 'N/A'
                const max = data.battery_soc_max?.toFixed(1) || 'N/A'
                
                return `Min: ${min}% | Mean: ${mean}% | Max: ${max}%`
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
              display: false
            },
            ticks: {
              display: false
            },
            grid: {
              display: true
            },
            stacked: true
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
              align: 'end',
              padding: 5,
              font: {
                family: 'monospace'
              },
              callback: function(value) {
                return value.toFixed(0)
              }
            },
            stacked: true
          }
        }
      }
    }
  }
}
</script>
