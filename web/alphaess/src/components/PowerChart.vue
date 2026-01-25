<template>
  <div style="position: relative; height: 400px;">
    <Bar :data="chartData" :options="chartOptions" />
  </div>
</template>

<script>
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  BarElement,
  LineElement,
  LineController,
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
  BarElement,
  LineElement,
  LineController,
  Title,
  Tooltip,
  Legend,
  TimeScale,
  Filler
)

export default {
  name: 'PowerChart',
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
      const colors = {
        'solar_production': { border: 'rgba(255, 193, 7, 0.5)', bg: 'rgba(255, 193, 7, 0.8)' },
        'solar_net': { border: 'rgba(255, 213, 79, 0.5)', bg: 'rgba(255, 213, 79, 0.8)' },
        'to_grid': { border: 'rgba(255, 179, 0, 0.5)', bg: 'rgba(255, 179, 0, 0.8)' },
        'battery_charge': { border: 'rgba(205, 220, 57, 0.5)', bg: 'rgba(205, 220, 57, 0.8)' },
        'from_grid': { border: 'rgba(244, 67, 54, 0.5)', bg: 'rgba(244, 67, 54, 0.8)' },
        'battery_discharge': { border: 'rgba(76, 175, 80, 0.5)', bg: 'rgba(76, 175, 80, 0.8)' },
        'battery_charge_from_grid': { border: 'rgba(156, 39, 176, 0.5)', bg: 'rgba(156, 39, 176, 0.8)' },
        'battery_soc': { border: 'rgba(139, 195, 74, 0.05)', bg: 'rgba(139, 195, 74, 0.2)' }
      }

      // Convert aggregate data to power data indexed by time
      const powerData = {}
      const allTimes = new Set()
      const gapFilledTimes = new Set()
      
      // Process aggregates (already in power format)
      this.measurements.forEach(agg => {
        // Use start_time as the timestamp for the hour (to position bar at hour start)
        const time = new Date(agg.start_time).getTime()
        allTimes.add(time)
        
        if (agg.gap) {
          gapFilledTimes.add(time)
        }
        
        powerData[time] = {
          solar_production: agg.solar_production || 0,
          to_grid: agg.to_grid || 0,
          from_grid: agg.from_grid || 0,
          battery_charge: agg.battery_charge || 0,
          battery_discharge: agg.battery_discharge || 0,
          battery_charge_from_grid: agg.battery_charge_from_grid || 0,
          battery_soc: agg.battery_soc || 0
        }
      })
      
      const allTimesSorted = Array.from(allTimes).sort((a, b) => a - b)

      // Calculate net solar production for each time point
      allTimesSorted.forEach(time => {
        const point = powerData[time]
        if (!point) return
        
        const solar = point.solar_production || 0
        const toGrid = point.to_grid || 0
        const batteryCharge = point.battery_charge || 0
        powerData[time].solar_net = Math.max(0, solar - toGrid - batteryCharge)

      })

      // Above axis (consumed power) - stacked, solar nearest to axis
      const aboveAxisOrder = [
        { name: 'solar_net', label: 'Solar Production (Net)', color: 'solar_net' },
        { name: 'battery_discharge', label: 'Battery Discharge', color: 'battery_discharge' },
        { name: 'from_grid', label: 'From Grid', color: 'from_grid' }
      ]
      
      // Create datasets with gap styling
      aboveAxisOrder.forEach(item => {
        const data = allTimesSorted.map(time => {
          const isGap = gapFilledTimes.has(time)
          const value = powerData[time][item.name] || 0
          
          // Adjust opacity for gaps
          const bgColor = isGap 
            ? colors[item.color].border.replace('0.5)', '0.2)').replace('0.8)', '0.3)')
            : colors[item.color].bg
          
          return {
            x: new Date(parseInt(time)),
            y: value,
            backgroundColor: bgColor
          }
        })
        
        datasets.push({
          label: item.label,
          data: data,
          borderColor: colors[item.color].border,
          backgroundColor: (context) => {
            return context.raw?.backgroundColor || colors[item.color].bg
          },
          borderWidth: 0,
          stack: 'power',
          yAxisID: 'y',
          type: 'bar',
          barPercentage: 0.9,
          categoryPercentage: 0.95
        })
      })

      // Below axis (unused power) - stacked, solar-related nearest to axis
      const belowAxisOrder = [
        { name: 'to_grid', label: 'To Grid', color: 'to_grid' },
        { name: 'battery_charge', label: 'Battery Charge', color: 'battery_charge' },
        { name: 'battery_charge_from_grid', label: 'Battery Charge From Grid', color: 'battery_charge_from_grid' }
      ]
      
      // Create datasets with gap styling
      belowAxisOrder.forEach(item => {
        const data = allTimesSorted.map(time => {
          const isGap = gapFilledTimes.has(time)
          const value = -(powerData[time][item.name] || 0)
          
          // Adjust opacity for gaps
          const bgColor = isGap 
            ? colors[item.color].border.replace('0.5)', '0.2)').replace('0.8)', '0.3)')
            : colors[item.color].bg
          
          return {
            x: new Date(parseInt(time)),
            y: value,
            backgroundColor: bgColor
          }
        })
        
        datasets.push({
          label: item.label,
          data: data,
          borderColor: colors[item.color].border,
          backgroundColor: (context) => {
            return context.raw?.backgroundColor || colors[item.color].bg
          },
          borderWidth: 0,
          stack: 'power',
          yAxisID: 'y',
          type: 'bar',
          barPercentage: 0.9,
          categoryPercentage: 0.95
        })
      })

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
                const value = Math.abs(context.parsed.y).toFixed(0)
                return `${label}: ${value} W`
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
            stacked: true
          },
          y: {
            type: 'linear',
            position: 'left',
            title: {
              display: true,
              text: 'Power (W)'
            },
            stacked: true,
            ticks: {
              callback: function(value) {
                return Math.abs(value).toFixed(0)
              }
            }
          }
        }
      }
    }
  },
  methods: {
    formatLabel(name) {
      return name
        .split('_')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ')
    }
  }
}
</script>
