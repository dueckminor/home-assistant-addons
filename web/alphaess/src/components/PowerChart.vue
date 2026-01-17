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
  TimeScale
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
  TimeScale
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
        'solar_production': { border: '#FF9800', bg: 'rgba(255, 152, 0, 0.1)' },
        'to_grid': { border: '#4CAF50', bg: 'rgba(76, 175, 80, 0.1)' },
        'from_grid': { border: '#2196F3', bg: 'rgba(33, 150, 243, 0.1)' },
        'battery_charge': { border: '#9C27B0', bg: 'rgba(156, 39, 176, 0.1)' },
        'battery_discharge': { border: '#E91E63', bg: 'rgba(233, 30, 99, 0.1)' },
        'battery_charge_from_grid': { border: '#00BCD4', bg: 'rgba(0, 188, 212, 0.1)' },
        'battery_soc': { border: '#8BC34A', bg: 'rgba(139, 195, 74, 0.1)' }
      }

      this.measurements.forEach(m => {
        if (colors[m.name] && m.values && m.values.length > 0) {
          const data = []
          
          // For accumulated energy (Wh), calculate power from consecutive values
          if (m.unit === 'Wh') {
            for (let i = 1; i < m.values.length; i++) {
              const current = m.values[i]
              const previous = m.values[i - 1]
              
              if (current && previous && current.time && previous.time) {
                const energyDiff = current.value - previous.value
                const timeDiff = (new Date(current.time) - new Date(previous.time)) / 1000 / 3600
                
                if (timeDiff > 0) {
                  const power = energyDiff / timeDiff
                  data.push({
                    x: new Date(current.time),
                    y: power
                  })
                }
              }
            }
          } else {
            // For direct measurements (like SOC %), use values directly
            m.values.forEach(v => {
              data.push({
                x: new Date(v.time),
                y: v.value
              })
            })
          }

          if (data.length > 0) {
            datasets.push({
              label: this.formatLabel(m.name),
              data: data,
              borderColor: colors[m.name].border,
              backgroundColor: colors[m.name].bg,
              borderWidth: 2,
              tension: 0.3,
              fill: true,
              pointRadius: 0,
              yAxisID: m.name === 'battery_soc' ? 'y1' : 'y'
            })
          }
        }
      })

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
                const value = context.parsed.y.toFixed(0)
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
            }
          },
          y: {
            type: 'linear',
            position: 'left',
            title: {
              display: true,
              text: 'Power (W)'
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
