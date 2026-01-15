import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Plugin to exclude heavy font formats (TTF, EOT, WOFF) - keep only WOFF2
const excludeHeavyFonts = () => ({
  name: 'exclude-heavy-fonts',
  generateBundle(options, bundle) {
    // Remove TTF, EOT, WOFF files from bundle, keep only WOFF2
    Object.keys(bundle).forEach(key => {
      if (/materialdesignicons-webfont.*\.(ttf|eot|woff)$/.test(key) && !/\.woff2$/.test(key)) {
        delete bundle[key]
      }
    })
  }
})

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue(), excludeHeavyFonts()],
  base: './',
  build: {
    outDir: '../../go/embed/mqtt_bridge_dist/dist',
    assetsDir: 'assets'
  },
  server: {
    port: 3003  ,
  },
  resolve: {
    alias: {
    '@': fileURLToPath(new URL('./src', import.meta.url))
    }
}
})