<template>
  <v-app>
    <!-- Header -->
    <v-app-bar color="primary" dark elevation="2">
      <v-icon class="me-3">mdi-gateway</v-icon>
      <v-toolbar-title>Gateway for Homeassistant</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn icon @click="showSettings = true">
        <v-icon>mdi-cog</v-icon>
      </v-btn>
    </v-app-bar>

    <!-- Settings Dialog -->
    <v-dialog v-model="showSettings" fullscreen transition="dialog-bottom-transition">
      <v-card>
        <v-toolbar color="primary" dark flat>
          <v-icon class="me-3">mdi-cog</v-icon>
          <v-toolbar-title>Settings</v-toolbar-title>
          <v-spacer />
          <v-btn icon @click="showSettings = false">
            <v-icon>mdi-close</v-icon>
          </v-btn>
        </v-toolbar>

        <v-tabs v-model="activeTab" bg-color="primary" slider-color="white">
          <v-tab value="dns">
            <v-icon start>mdi-dns</v-icon>
            DNS
          </v-tab>
          <v-tab value="domains">
            <v-icon start>mdi-sitemap</v-icon>
            Domains &amp; Routes
          </v-tab>
          <v-tab value="users">
            <v-icon start>mdi-account-group</v-icon>
            Users
          </v-tab>
          <v-tab value="mail">
            <v-icon start>mdi-email</v-icon>
            Mail
          </v-tab>
        </v-tabs>

        <v-card-text class="pa-6">
          <v-tabs-window v-model="activeTab">
            <v-tabs-window-item value="dns">
              <DnsTab
                :dns-config="dnsConfig"
                :ip-detection-methods="ipDetectionMethods"
                :ipv6-detection-methods="ipv6DetectionMethods"
                :ipv4-loading="ipv4Loading"
                :ipv6-loading="ipv6Loading"
                :ipv4-testing="ipv4Testing"
                :ipv6-testing="ipv6Testing"
                :show-ipv4-revert="showIPv4RevertButton"
                :show-ipv6-revert="showIPv6RevertButton"
                @refresh-ipv4="refreshIPv4"
                @refresh-ipv6="refreshIPv6"
                @test-ipv4="testIPv4Config"
                @test-ipv6="testIPv6Config"
                @revert-ipv4="revertIPv4Source"
                @revert-ipv6="revertIPv6Source"
              />
            </v-tabs-window-item>

            <v-tabs-window-item value="domains">
              <DomainsTab />
            </v-tabs-window-item>

            <v-tabs-window-item value="users">
              <UsersTab :user-config="userConfig" />
            </v-tabs-window-item>

            <v-tabs-window-item value="mail">
              <MailTab />
            </v-tabs-window-item>
          </v-tabs-window>
        </v-card-text>
      </v-card>
    </v-dialog>

    <!-- Main Content: Metrics always visible -->
    <v-main>
      <v-container fluid class="pa-4">
        <MetricsTab :is-active="true" />
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import DnsTab from '../components/tabs/DnsTab.vue'
import DomainsTab from '../components/tabs/DomainsTab.vue'
import UsersTab from '../components/tabs/UsersTab.vue'
import MailTab from '../components/tabs/MailTab.vue'
import MetricsTab from '../components/tabs/MetricsTab.vue'
import { apiGet, apiPost } from '../../../shared/utils/homeassistant.js'

export default {
  name: 'Index',
  components: {
    DnsTab,
    DomainsTab,
    UsersTab,
    MailTab,
    MetricsTab
  },
  data() {
    return {
      showSettings: false,
      activeTab: 'dns',
      ipv4Loading: false,
      ipv6Loading: false,
      ipv4Testing: false,
      ipv6Testing: false,
      ipv4Timeout: null,
      ipv6Timeout: null,
      initialLoadingComplete: false,

      lastValidIPv4Source: '',
      lastValidIPv6Source: '',

      ipDetectionMethods: [
        { title: 'DNS', value: 'dns' }
      ],

      ipv6DetectionMethods: [
        { title: 'DNS', value: 'dns' },
        { title: 'Home-Assistant', value: 'homeassistant' }
      ],

      dnsConfig: {
        ipv4: { method: 'dns', source: '', current: '', lastUpdate: null },
        ipv6: { method: 'dns', source: '', current: '', lastUpdate: null }
      },

      userConfig: {}
    }
  },
  computed: {
    showIPv4RevertButton() {
      return this.dnsConfig.ipv4.current?.startsWith('Error:') &&
             this.lastValidIPv4Source &&
             this.lastValidIPv4Source !== this.dnsConfig.ipv4.source
    },
    showIPv6RevertButton() {
      return this.dnsConfig.ipv6.method === 'dns' &&
             this.dnsConfig.ipv6.current?.startsWith('Error:') &&
             this.lastValidIPv6Source &&
             this.lastValidIPv6Source !== this.dnsConfig.ipv6.source
    }
  },
  async mounted() {
    await Promise.all([
      this.loadConfiguration(),
      this.getInitialExternalIPv4(),
      this.getInitialExternalIPv6()
    ])
    this.initialLoadingComplete = true
  },
  watch: {
    'dnsConfig.ipv4.source'(newValue, oldValue) {
      if (newValue !== oldValue && newValue.trim() && this.initialLoadingComplete) {
        if (this.ipv4Timeout) clearTimeout(this.ipv4Timeout)
        this.ipv4Timeout = setTimeout(async () => {
          const isValid = await this.validateAndUpdateIPv4Config(newValue)
          if (isValid) this.refreshIPv4()
        }, 500)
      }
    },
    'dnsConfig.ipv6.source'(newValue, oldValue) {
      if (this.dnsConfig.ipv6.method !== 'dns') return
      if (newValue !== oldValue && newValue.trim() && this.initialLoadingComplete) {
        if (this.ipv6Timeout) clearTimeout(this.ipv6Timeout)
        this.ipv6Timeout = setTimeout(async () => {
          const isValid = await this.validateAndUpdateIPv6Config(newValue)
          if (isValid) this.refreshIPv6()
        }, 500)
      }
    },
    'dnsConfig.ipv6.method'(newMethod, oldMethod) {
      if (newMethod !== oldMethod && this.initialLoadingComplete) {
        if (newMethod === 'homeassistant') {
          this.validateAndUpdateIPv6Config('').then(isValid => {
            if (isValid) this.refreshIPv6()
          })
        }
      }
    }
  },
  beforeUnmount() {
    if (this.ipv4Timeout) clearTimeout(this.ipv4Timeout)
    if (this.ipv6Timeout) clearTimeout(this.ipv6Timeout)
  },
  methods: {
    async getInitialExternalIPv4() {
      try {
        const data = await apiGet('dns/external/ipv4')
        if (data) {
          this.dnsConfig.ipv4.source = data.param || ''
          this.dnsConfig.ipv4.current = data.error ? `Error: ${data.error}` : (data.address || '')
          this.dnsConfig.ipv4.method = data.method || 'dns'
          this.dnsConfig.ipv4.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : null
          if (!data.error && data.param && data.address) this.lastValidIPv4Source = data.param
        }
      } catch { /* ignore */ }
    },

    async getInitialExternalIPv6() {
      try {
        const data = await apiGet('dns/external/ipv6')
        if (data) {
          this.dnsConfig.ipv6.source = data.param || ''
          this.dnsConfig.ipv6.current = data.error ? `Error: ${data.error}` : (data.address || '')
          this.dnsConfig.ipv6.method = data.method || 'dns'
          this.dnsConfig.ipv6.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : null
          if (!data.error && data.method === 'dns' && data.param) this.lastValidIPv6Source = data.param
        }
      } catch { /* ignore */ }
    },

    async validateAndUpdateIPv4Config(sourceAddress) {
      try {
        const testData = await apiPost('dns/external/ipv4', { method: 'dns', param: sourceAddress, test: true })
        if (testData?.address && !testData.error) {
          await this.updateExternalIPv4Config(sourceAddress)
          this.lastValidIPv4Source = sourceAddress
          return true
        }
        this.dnsConfig.ipv4.current = `Error: ${testData.error || 'DNS resolution failed'}`
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString()
        return false
      } catch {
        this.dnsConfig.ipv4.current = 'Network Error'
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString()
        return false
      }
    },

    async validateAndUpdateIPv6Config(sourceAddress) {
      try {
        const method = this.dnsConfig.ipv6.method
        const payload = { method, test: true }
        if (method === 'dns') payload.param = sourceAddress
        const testData = await apiPost('dns/external/ipv6', payload)
        if (testData?.address && !testData.error) {
          await this.updateExternalIPv6Config(sourceAddress)
          if (method === 'dns') this.lastValidIPv6Source = sourceAddress
          return true
        }
        this.dnsConfig.ipv6.current = `Error: ${testData.error || 'Configuration failed'}`
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString()
        return false
      } catch {
        this.dnsConfig.ipv6.current = 'Network Error'
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString()
        return false
      }
    },

    async updateExternalIPv4Config(sourceAddress) {
      try {
        await apiPost('dns/external/ipv4', { method: 'dns', param: sourceAddress })
      } catch { /* ignore */ }
    },

    async updateExternalIPv6Config(sourceAddress) {
      try {
        const method = this.dnsConfig.ipv6.method
        const payload = { method }
        if (method === 'dns') payload.param = sourceAddress
        await apiPost('dns/external/ipv6', payload)
      } catch { /* ignore */ }
    },

    async revertIPv4Source() {
      if (this.lastValidIPv4Source) this.dnsConfig.ipv4.source = this.lastValidIPv4Source
    },

    async revertIPv6Source() {
      if (this.lastValidIPv6Source) this.dnsConfig.ipv6.source = this.lastValidIPv6Source
    },

    async refreshIPv4() {
      this.ipv4Loading = true
      try {
        const data = await apiGet('dns/external/ipv4')
        if (data) {
          this.dnsConfig.ipv4.current = data.error ? `Error: ${data.error}` : (data.address || '')
          this.dnsConfig.ipv4.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : new Date().toLocaleString()
        }
      } catch {
        this.dnsConfig.ipv4.current = 'Network Error'
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString()
      } finally {
        this.ipv4Loading = false
      }
    },

    async refreshIPv6() {
      this.ipv6Loading = true
      try {
        const data = await apiGet('dns/external/ipv6')
        if (data) {
          this.dnsConfig.ipv6.current = data.error ? `Error: ${data.error}` : (data.address || '')
          this.dnsConfig.ipv6.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : new Date().toLocaleString()
        }
      } catch {
        this.dnsConfig.ipv6.current = 'Network Error'
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString()
      } finally {
        this.ipv6Loading = false
      }
    },

    async loadConfiguration() {
      await Promise.all([this.refreshIPv4(), this.refreshIPv6()])
    },

    async testIPv4Config() {
      if (!this.dnsConfig.ipv4.source.trim()) return
      this.ipv4Testing = true
      try {
        const testData = await apiPost('dns/external/ipv4', { method: 'dns', param: this.dnsConfig.ipv4.source, test: true })
        const orig = this.dnsConfig.ipv4.current
        this.dnsConfig.ipv4.current = testData?.address && !testData.error
          ? `Test: ${testData.address} ✓`
          : `Test Error: ${testData.error || 'Unknown error'} ✗`
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString()
        setTimeout(() => {
          if (this.dnsConfig.ipv4.current.startsWith('Test')) this.dnsConfig.ipv4.current = orig
        }, testData?.address && !testData.error ? 3000 : 5000)
      } catch {
        const orig = this.dnsConfig.ipv4.current
        this.dnsConfig.ipv4.current = 'Test Network Error ✗'
        setTimeout(() => { if (this.dnsConfig.ipv4.current === 'Test Network Error ✗') this.dnsConfig.ipv4.current = orig }, 5000)
      } finally {
        this.ipv4Testing = false
      }
    },

    async testIPv6Config() {
      const method = this.dnsConfig.ipv6.method
      if (method === 'dns' && !this.dnsConfig.ipv6.source.trim()) return
      this.ipv6Testing = true
      try {
        const payload = { method, test: true }
        if (method === 'dns') payload.param = this.dnsConfig.ipv6.source
        const testData = await apiPost('dns/external/ipv6', payload)
        const orig = this.dnsConfig.ipv6.current
        this.dnsConfig.ipv6.current = testData?.address && !testData.error
          ? `Test: ${testData.address} ✓`
          : `Test Error: ${testData.error || 'Unknown error'} ✗`
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString()
        setTimeout(() => {
          if (this.dnsConfig.ipv6.current.startsWith('Test')) this.dnsConfig.ipv6.current = orig
        }, testData?.address && !testData.error ? 3000 : 5000)
      } catch {
        const orig = this.dnsConfig.ipv6.current
        this.dnsConfig.ipv6.current = 'Test Network Error ✗'
        setTimeout(() => { if (this.dnsConfig.ipv6.current === 'Test Network Error ✗') this.dnsConfig.ipv6.current = orig }, 5000)
      } finally {
        this.ipv6Testing = false
      }
    }
  }
}
</script>
