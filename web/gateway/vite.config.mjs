import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import { fileURLToPath, URL } from 'node:url'
import { copyFileSync, existsSync, mkdirSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'

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

// maplibre-gl v6 splits into main, worker, and shared modules. Vite can't statically
// analyze the dynamic worker URL, so none of the runtime files end up in the build.
// Copy all non-dev .mjs files from maplibre-gl/dist to the assets dir manually.
const copyMaplibreWorker = () => {
  let assetsDir
  return {
    name: 'copy-maplibre-worker',
    apply: 'build',
    configResolved(config) {
      assetsDir = resolve(config.build.outDir, config.build.assetsDir || 'assets')
    },
    closeBundle() {
      const srcDir = fileURLToPath(new URL('./node_modules/maplibre-gl/dist', import.meta.url))
      if (!existsSync(assetsDir)) mkdirSync(assetsDir, { recursive: true })
      for (const file of readdirSync(srcDir)) {
        if (file.endsWith('.mjs') && !file.includes('-dev.')) {
          copyFileSync(resolve(srcDir, file), resolve(assetsDir, file))
        }
      }
    }
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue({
      template: { transformAssetUrls }
    }),
    vuetify({
      autoImport: true,
    }),
    excludeHeavyFonts(),
    copyMaplibreWorker()
  ],
  base: './', // Generate relative paths instead of absolute
  build: {
    outDir: '../../go/embed/gateway_dist/dist',
    assetsDir: 'assets'
  },
  server: {
    port: 3001
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  optimizeDeps: {
    exclude: ['maplibre-gl']
  }
})