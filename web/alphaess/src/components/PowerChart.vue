<template>
  <div style="position: relative; height: 400px;">
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
  name: 'PowerChart',
  components: {
    Line
  },
  props: {
    measurements: {
      type: Array,
      required: true
    }
  },
  computed: {
    chartData() {
      const datasets = []
      const colors = {
        'solar_production': { border: 'rgba(255, 193, 7, 0.5)', bg: 'rgba(255, 193, 7, 0.8)' },
        'solar_net': { border: 'rgba(255, 213, 79, 0.5)', bg: 'rgba(255, 213, 79, 0.8)' },
        'to_grid': { border: 'rgba(255, 179, 0, 0.5)', bg: 'rgba(255, 179, 0, 0.8)' },
        'battery_charge': { border: 'rgba(255, 167, 38, 0.5)', bg: 'rgba(255, 167, 38, 0.8)' },
        'from_grid': { border: 'rgba(244, 67, 54, 0.5)', bg: 'rgba(244, 67, 54, 0.8)' },
        'battery_discharge': { border: 'rgba(76, 175, 80, 0.5)', bg: 'rgba(76, 175, 80, 0.8)' },
        'battery_charge_from_grid': { border: 'rgba(156, 39, 176, 0.5)', bg: 'rgba(156, 39, 176, 0.8)' },
        'battery_soc': { border: 'rgba(139, 195, 74, 0.05)', bg: 'rgba(139, 195, 74, 0.2)' }
      }

      // Convert all measurements to power data indexed by time
      const powerData = {}
      const allTimes = new Set()
      
      this.measurements.forEach(m => {
        if (m.values && m.values.length > 0 && m.unit === 'Wh') {
          for (let i = 1; i < m.values.length; i++) {
            const current = m.values[i]
            const previous = m.values[i - 1]
            
            if (current && previous && current.time && previous.time) {
              const energyDiff = current.value - previous.value
              const timeDiff = (new Date(current.time) - new Date(previous.time)) / 1000 / 3600
              
              if (timeDiff > 0) {
                const power = energyDiff / timeDiff
                const time = new Date(current.time).getTime()
                allTimes.add(time)
                
                if (!powerData[time]) {
                  powerData[time] = {}
                }
                powerData[time][m.name] = power
              }
            }
          }
        } else if (m.name === 'battery_soc' && m.values && m.values.length > 0) {
          // Handle SOC separately (not Wh)
          m.values.forEach(v => {
            const time = new Date(v.time).getTime()
            allTimes.add(time)
            if (!powerData[time]) {
              powerData[time] = {}
            }
            powerData[time][m.name] = v.value
          })
        }
      })

      const sortedTimes = Array.from(allTimes).sort((a, b) => a - b)

      // Fill in missing values with 0 for proper stacking
      sortedTimes.forEach(time => {
        if (!powerData[time]) {
          powerData[time] = {}
        }
      })

      // Calculate net solar production for each time point
      sortedTimes.forEach(time => {
        const point = powerData[time]
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
      
      aboveAxisOrder.forEach(item => {
        const data = sortedTimes.map(time => ({
          x: new Date(parseInt(time)),
          y: powerData[time][item.name] || 0
        }))
        
        datasets.push({
          label: item.label,
          data: data,
          borderColor: colors[item.color].border,
          backgroundColor: colors[item.color].bg,
          borderWidth: 1,
          tension: 0,
          fill: 'origin',
          pointRadius: 0,
          stack: 'consumed',
          yAxisID: 'y'
        })
      })

      // Below axis (unused power) - stacked, solar-related nearest to axis
      const belowAxisOrder = [
        { name: 'to_grid', label: 'To Grid', color: 'to_grid' },
        { name: 'battery_charge', label: 'Battery Charge', color: 'battery_charge' },
        { name: 'battery_charge_from_grid', label: 'Battery Charge From Grid', color: 'battery_charge_from_grid' }
      ]
      
      belowAxisOrder.forEach(item => {
        const data = sortedTimes.map(time => ({
          x: new Date(parseInt(time)),
          y: -(powerData[time][item.name] || 0)
        }))
        
        datasets.push({
          label: item.label,
          data: data,
          borderColor: colors[item.color].border,
          backgroundColor: colors[item.color].bg,
          borderWidth: 1,
          tension: 0,
          fill: 'origin',
          pointRadius: 0,
          stack: 'unused',
          yAxisID: 'y'
        })
      })

      // Add battery SOC in background (not stacked, separate axis)
      const socData = sortedTimes
        .filter(time => powerData[time].battery_soc !== undefined)
        .map(time => ({
          x: new Date(parseInt(time)),
          y: powerData[time].battery_soc
        }))
      
      if (socData.length > 0) {
        // Add SOC as first dataset (background)
        datasets.unshift({
          label: 'Battery SOC',
          data: socData,
          borderColor: colors.battery_soc.border,
          backgroundColor: colors.battery_soc.bg,
          borderWidth: 1,
          tension: 0.3,
          fill: true,
          pointRadius: 0,
          yAxisID: 'y1',
          order: -1  // Draw first (background)
        })
      }

      return { datasets }
    },
    chartOptions() {
      return {
        responsive: true,
        maintainAspectRatio: false,
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
                const unit = context.dataset.yAxisID === 'y1' ? '%' : 'W'
                return `${label}: ${value} ${unit}`
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
            title: {
              display: true,
              text: 'Time'
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
          },
          y1: {
            type: 'linear',
            position: 'right',
            min: 0,
            max: 100,
            title: {
              display: true,
              text: 'SOC (%)'
            },
            grid: {
              drawOnChartArea: false
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
