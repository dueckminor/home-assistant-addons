import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import 'vuetify/styles'
import './styles/mdi-optimized.css'

import App from './App.vue'

const vuetify = createVuetify({
  theme: {
    defaultTheme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
})

const app = createApp(App)
app.use(vuetify)
app.mount('#app')