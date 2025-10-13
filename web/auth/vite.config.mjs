// Plugins
import Components from 'unplugin-vue-components/vite'
import Vue from '@vitejs/plugin-vue'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import ViteFonts from 'unplugin-fonts/vite'
import VueRouter from 'unplugin-vue-router/vite'

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import { copyFileSync, existsSync, mkdirSync } from 'fs'
import { resolve, dirname } from 'path'

// Plugin to copy WOFF2 font from our own node_modules during build
const copyMdiFont = () => ({
  name: 'copy-mdi-font',
  buildStart() {
    // Source: our own node_modules
    const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), 'node_modules/@mdi/font/fonts/materialdesignicons-webfont.woff2')
    // Target: auth public assets
    const targetPath = resolve(dirname(fileURLToPath(import.meta.url)), 'public/assets/materialdesignicons-webfont.woff2')
    
    if (existsSync(sourcePath)) {
      console.log('Copying MDI WOFF2 font from auth node_modules...')
      // Ensure target directory exists
      const targetDir = dirname(targetPath)
      if (!existsSync(targetDir)) {
        mkdirSync(targetDir, { recursive: true })
      }
      copyFileSync(sourcePath, targetPath)
    } else {
      console.warn('MDI WOFF2 font not found in auth node_modules')
    }
  }
})

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    copyMdiFont(),
    VueRouter({
      routesFolder: 'src/pages',
      extensions: ['.vue'],
      dts: './typed-router.d.ts'
    }),
    Vue({
      template: { transformAssetUrls }
    }),
    // https://github.com/vuetifyjs/vuetify-loader/tree/master/packages/vite-plugin#readme
    Vuetify({
      autoImport: true,
      styles: {
        configFile: 'src/styles/settings.scss',
      },
    }),
    Components(),
    ViteFonts({
      google: {
        families: [{
          name: 'Roboto',
          styles: 'wght@100;300;400;500;700;900',
        }],
      },
    }),
  ],
  build: {
    outDir: '../../go/auth/dist',
  },
  base: './', // Generate relative paths instead of absolute
  define: { 'process.env': {} },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
    extensions: [
      '.js',
      '.json',
      '.jsx',
      '.mjs',
      '.ts',
      '.tsx',
      '.vue',
    ],
  },
  server: {
    port: 3000,
    allowedHosts: ['auth.rh94-dev.dueckminor.de']
  },
})
