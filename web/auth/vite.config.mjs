// Plugins
import Components from 'unplugin-vue-components/vite'
import Vue from '@vitejs/plugin-vue'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import ViteFonts from 'unplugin-fonts/vite'
import VueRouter from 'unplugin-vue-router/vite'

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import { copyFileSync, existsSync, mkdirSync, readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import yaml from 'js-yaml'

// Function to calculate allowed hosts from gateway config
const getAuthAllowedHosts = () => {
  try {
    const configPath = resolve(dirname(fileURLToPath(import.meta.url)), '../../gen/data/gateway/config.yml')
    if (!existsSync(configPath)) {
      console.warn('Gateway config not found, using localhost for dev server')
      return ['localhost']
    }
    
    const configContent = readFileSync(configPath, 'utf8')
    const config = yaml.load(configContent)
    
    const allowedHosts = []
    
    // Find all routes with target "@auth" across all domains
    if (config.domains && Array.isArray(config.domains)) {
      for (const domain of config.domains) {
        if (domain.routes && Array.isArray(domain.routes)) {
          for (const route of domain.routes) {
            if (route.target === '@auth' && route.hostname && domain.name) {
              allowedHosts.push(`${route.hostname}.${domain.name}`)
            }
          }
        }
      }
    }
    
    // Always include localhost for local development
    if (!allowedHosts.includes('localhost')) {
      allowedHosts.push('localhost')
    }
    
    console.log('Auth allowed hosts:', allowedHosts)
    return allowedHosts
  } catch (error) {
    console.warn('Failed to read gateway config:', error.message)
    return ['localhost']
  }
}

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
    allowedHosts: getAuthAllowedHosts()
  },
})
