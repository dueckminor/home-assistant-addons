import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  base: './', // Generate relative paths instead of absolute
  build: {
    outDir: '../../go/gateway/dist',
    assetsDir: 'assets'
  },
  server: {
    port: 3001
  }
})