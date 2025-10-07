<template>
  <v-app>
    <!-- Header -->
    <v-app-bar color="primary" dark elevation="2">
      <v-icon class="me-3">mdi-gateway</v-icon>
      <v-toolbar-title>Gateway Configuration</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-chip color="success" variant="outlined">
        <v-icon start>mdi-shield-check</v-icon>
        Secure Access
      </v-chip>
    </v-app-bar>

    <!-- Main Content -->
    <v-main>
      <v-container fluid class="pa-4">
        <v-row>
          <v-col cols="12">
            <!-- Navigation Tabs -->
            <v-card>
              <v-tabs v-model="activeTab" bg-color="primary" slider-color="white">
                <v-tab value="dns">
                  <v-icon start>mdi-dns</v-icon>
                  DNS
                </v-tab>
                <v-tab value="routes">
                  <v-icon start>mdi-routes</v-icon>
                  Routes
                </v-tab>
                <v-tab value="users">
                  <v-icon start>mdi-account-group</v-icon>
                  Users
                </v-tab>
                <v-tab value="certificates">
                  <v-icon start>mdi-certificate</v-icon>
                  Certificates
                </v-tab>
              </v-tabs>

              <v-card-text class="pa-6">
                <v-tabs-window v-model="activeTab">
                  <!-- DNS Tab -->
                  <v-tabs-window-item value="dns">
                    <DnsTab 
                      :dns-config="dnsConfig"
                      :ip-detection-methods="ipDetectionMethods"
                      :ipv4-loading="ipv4Loading"
                      :ipv6-loading="ipv6Loading"
                      @refresh-ipv4="refreshIPv4"
                      @refresh-ipv6="refreshIPv6"
                      @save-config="saveConfiguration"
                    />
                  </v-tabs-window-item>

                  <!-- Routes Tab -->
                  <v-tabs-window-item value="routes">
                    <RoutesTab 
                      :route-config="routeConfig"
                      @save-config="saveConfiguration"
                    />
                  </v-tabs-window-item>

                  <!-- Users Tab -->
                  <v-tabs-window-item value="users">
                    <UsersTab 
                      :user-config="userConfig"
                      @save-config="saveConfiguration"
                    />
                  </v-tabs-window-item>

                  <!-- Certificates Tab -->
                  <v-tabs-window-item value="certificates">
                    <CertificatesTab 
                      :certificate-config="certificateConfig"
                      @save-config="saveConfiguration"
                    />
                  </v-tabs-window-item>
                </v-tabs-window>
              </v-card-text>
            </v-card>

            <!-- Status Footer -->
            <v-card class="mt-6" color="grey-lighten-4">
              <v-card-text class="text-center pa-4">
                <v-chip color="success" class="ma-1">
                  <v-icon start>mdi-server</v-icon>
                  Gateway: Running
                </v-chip>
                <v-chip color="info" class="ma-1">
                  <v-icon start>mdi-network</v-icon>
                  Ports: 53, 80, 443, 8099
                </v-chip>
                <v-chip color="warning" class="ma-1">
                  <v-icon start>mdi-construction</v-icon>
                  Configuration: In Development
                </v-chip>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import DnsTab from '../components/tabs/DnsTab.vue'
import RoutesTab from '../components/tabs/RoutesTab.vue'
import UsersTab from '../components/tabs/UsersTab.vue'
import CertificatesTab from '../components/tabs/CertificatesTab.vue'

export default {
  name: 'Index',
  components: {
    DnsTab,
    RoutesTab,
    UsersTab,
    CertificatesTab
  },
  data() {
    return {
      activeTab: 'dns', // Default to DNS tab
      ipv4Loading: false,
      ipv6Loading: false,
      ipv4Timeout: null,
      ipv6Timeout: null,
      
      // IP Detection Methods (expandable for future methods)
      ipDetectionMethods: [
        { title: 'DNS', value: 'dns' }
      ],
      
      // DNS Configuration
      dnsConfig: {
        ipv4: {
          method: 'dns',
          source: '',
          current: '',
          lastUpdate: null
        },
        ipv6: {
          method: 'dns', 
          source: '',
          current: '',
          lastUpdate: null
        },
        domains: []
      },
      
      // Future configuration data
      routeConfig: {},
      userConfig: {},
      certificateConfig: {}
    }
  },
  mounted() {
    console.log('Gateway Configuration UI loaded')
    console.log('Active tab:', this.activeTab)
    
    // Load initial configuration
    this.loadConfiguration()
  },

  watch: {
    // Auto-refresh IPv4 when source address changes (debounced)
    'dnsConfig.ipv4.source'(newValue, oldValue) {
      if (newValue !== oldValue && newValue.trim()) {
        console.log('IPv4 source changed:', newValue)
        
        // Clear existing timeout
        if (this.ipv4Timeout) {
          clearTimeout(this.ipv4Timeout)
        }
        
        // Set new timeout for debounced refresh
        this.ipv4Timeout = setTimeout(() => {
          console.log('Auto-refreshing IPv4...')
          this.refreshIPv4()
        }, 500) // 500ms debounce
      }
    },

    // Auto-refresh IPv6 when source address changes (debounced)
    'dnsConfig.ipv6.source'(newValue, oldValue) {
      if (newValue !== oldValue && newValue.trim()) {
        console.log('IPv6 source changed:', newValue)
        
        // Clear existing timeout
        if (this.ipv6Timeout) {
          clearTimeout(this.ipv6Timeout)
        }
        
        // Set new timeout for debounced refresh
        this.ipv6Timeout = setTimeout(() => {
          console.log('Auto-refreshing IPv6...')
          this.refreshIPv6()
        }, 500) // 500ms debounce
      }
    }
  },
  methods: {
    // IP Address refresh methods
    async refreshIPv4() {
      if (!this.dnsConfig.ipv4.source.trim()) {
        console.warn('No IPv4 source address configured');
        return;
      }

      this.ipv4Loading = true;
      try {
        const response = await fetch(`/api/dns/ipv4/${encodeURIComponent(this.dnsConfig.ipv4.source)}`);
        const data = await response.json();
        
        if (response.ok) {
          this.dnsConfig.ipv4.current = data.ip;
          this.dnsConfig.ipv4.lastUpdate = new Date(data.timestamp).toLocaleString();
          console.log('IPv4 refreshed:', this.dnsConfig.ipv4.current);
        } else {
          console.error('DNS lookup failed:', data.error);
          this.dnsConfig.ipv4.current = `Error: ${data.error}`;
          this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString();
        }
      } catch (error) {
        console.error('Failed to refresh IPv4:', error);
        this.dnsConfig.ipv4.current = 'Network Error';
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString();
      } finally {
        this.ipv4Loading = false;
      }
    },

    async refreshIPv6() {
      if (!this.dnsConfig.ipv6.source.trim()) {
        console.warn('No IPv6 source address configured');
        return;
      }

      this.ipv6Loading = true;
      try {
        const response = await fetch(`/api/dns/ipv6/${encodeURIComponent(this.dnsConfig.ipv6.source)}`);
        const data = await response.json();
        
        if (response.ok) {
          this.dnsConfig.ipv6.current = data.ip;
          this.dnsConfig.ipv6.lastUpdate = new Date(data.timestamp).toLocaleString();
          console.log('IPv6 refreshed:', this.dnsConfig.ipv6.current);
        } else {
          console.error('DNS lookup failed:', data.error);
          this.dnsConfig.ipv6.current = `Error: ${data.error}`;
          this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString();
        }
      } catch (error) {
        console.error('Failed to refresh IPv6:', error);
        this.dnsConfig.ipv6.current = 'Network Error';
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString();
      } finally {
        this.ipv6Loading = false;
      }
    },



    // Configuration management
    async loadConfiguration() {
      try {
        console.log('Loading configuration...');
        
        const response = await fetch('/api/config');
        if (response.ok) {
          const config = await response.json();
          
          // Load DNS configuration from API
          if (config.external_ip) {
            this.dnsConfig.ipv4.method = config.external_ip.source || 'dns';
            this.dnsConfig.ipv4.source = config.external_ip.options || '';
          }
          
          if (config.external_ipv6) {
            this.dnsConfig.ipv6.method = config.external_ipv6.source || 'dns';
            this.dnsConfig.ipv6.source = config.external_ipv6.options || '';
          }
          
          if (config.domains && Array.isArray(config.domains)) {
            this.dnsConfig.domains = [...config.domains];
          }
          
          console.log('Configuration loaded:', config);
        } else {
          console.warn('Failed to load configuration, using defaults');
          // Set reasonable defaults
          this.dnsConfig.ipv4.source = '';
          this.dnsConfig.ipv6.source = '';
        }
        
        // Auto-refresh IPs after loading configuration
        const refreshPromises = [];
        if (this.dnsConfig.ipv4.source) {
          refreshPromises.push(this.refreshIPv4());
        }
        if (this.dnsConfig.ipv6.source) {
          refreshPromises.push(this.refreshIPv6());
        }
        
        if (refreshPromises.length > 0) {
          await Promise.all(refreshPromises);
        }
      } catch (error) {
        console.error('Failed to load configuration:', error);
      }
    },

    async saveConfiguration() {
      try {
        console.log('Saving configuration...', this.dnsConfig);
        
        const configData = {
          external_ip: {
            source: this.dnsConfig.ipv4.method,
            options: this.dnsConfig.ipv4.source
          },
          external_ipv6: {
            source: this.dnsConfig.ipv6.method,
            options: this.dnsConfig.ipv6.source
          },
          domains: [...this.dnsConfig.domains]
        };
        
        const response = await fetch('/api/config', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(configData)
        });
        
        if (response.ok) {
          console.log('Configuration saved successfully');
          // TODO: Show success message to user
        } else {
          const error = await response.text();
          console.error('Failed to save configuration:', error);
          // TODO: Show error message to user
        }
      } catch (error) {
        console.error('Failed to save configuration:', error);
        // TODO: Show error message to user
      }
    }
  },

  beforeUnmount() {
    // Clean up timeouts to prevent memory leaks
    if (this.ipv4Timeout) {
      clearTimeout(this.ipv4Timeout)
    }
    if (this.ipv6Timeout) {
      clearTimeout(this.ipv6Timeout)
    }
  }
}
</script>

<style scoped>
/* Custom styles for the main layout */
</style>