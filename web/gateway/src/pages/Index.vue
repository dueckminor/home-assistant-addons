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
                <v-tab value="domains">
                  <v-icon start>mdi-sitemap</v-icon>
                  Domains & Routes
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
                  <!-- DNS Tab -->
                  <v-tabs-window-item value="dns">
                    <DnsTab 
                      :dns-config="dnsConfig"
                      :ip-detection-methods="ipDetectionMethods"
                      :ipv4-loading="ipv4Loading"
                      :ipv6-loading="ipv6Loading"
                      :show-ipv4-revert="showIPv4RevertButton"
                      :show-ipv6-revert="showIPv6RevertButton"
                      @refresh-ipv4="refreshIPv4"
                      @refresh-ipv6="refreshIPv6"
                      @revert-ipv4="revertIPv4Source"
                      @revert-ipv6="revertIPv6Source"
                    />
                  </v-tabs-window-item>

                  <!-- Domains & Routes Tab -->
                  <v-tabs-window-item value="domains">
                    <DomainsTab />
                  </v-tabs-window-item>

                  <!-- Users Tab -->
                  <v-tabs-window-item value="users">
                    <UsersTab 
                      :user-config="userConfig"
                    />
                  </v-tabs-window-item>

                  <!-- Mail Configuration Tab -->
                  <v-tabs-window-item value="mail">
                    <MailTab />
                  </v-tabs-window-item>
                </v-tabs-window>
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
import DomainsTab from '../components/tabs/DomainsTab.vue'
import UsersTab from '../components/tabs/UsersTab.vue'
import MailTab from '../components/tabs/MailTab.vue'
import { apiRequest, apiGet, apiPost } from '../utils/api.js'

export default {
  name: 'Index',
  components: {
    DnsTab,
    DomainsTab,
    UsersTab,
    MailTab
  },
  data() {
    return {
      activeTab: 'dns', // Default to DNS tab
      ipv4Loading: false,
      ipv6Loading: false,
      ipv4Timeout: null,
      ipv6Timeout: null,
      initialLoadingComplete: false, // Track initial loading state
      
      // Last valid configurations for revert functionality
      lastValidIPv4Source: '',
      lastValidIPv6Source: '',
      
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
        }
      },
      
      // Future configuration data
      userConfig: {}
    }
  },
  computed: {
    // Show revert button if current address shows an error and we have a valid fallback
    showIPv4RevertButton() {
      return this.dnsConfig.ipv4.current?.startsWith('Error:') && 
             this.lastValidIPv4Source && 
             this.lastValidIPv4Source !== this.dnsConfig.ipv4.source;
    },
    
    showIPv6RevertButton() {
      return this.dnsConfig.ipv6.current?.startsWith('Error:') && 
             this.lastValidIPv6Source && 
             this.lastValidIPv6Source !== this.dnsConfig.ipv6.source;
    }
  },
  async mounted() {
    console.log('Gateway Configuration UI loaded')
    console.log('Active tab:', this.activeTab)
    
    // Load initial configuration and external IPs
    await Promise.all([
      this.loadConfiguration(),
      this.getInitialExternalIPv4(),
      this.getInitialExternalIPv6()
    ])
    
    // Mark initial loading as complete
    this.initialLoadingComplete = true
    console.log('Initial loading complete')
  },

  watch: {
    // Validate and update IPv4 configuration when source address changes (debounced)
    'dnsConfig.ipv4.source'(newValue, oldValue) {
      if (newValue !== oldValue && newValue.trim() && this.initialLoadingComplete) {
        console.log('IPv4 source changed:', newValue)
        
        // Clear existing timeout
        if (this.ipv4Timeout) {
          clearTimeout(this.ipv4Timeout)
        }
        
        // Set new timeout for debounced validation and configuration update
        this.ipv4Timeout = setTimeout(async () => {
          console.log('Validating IPv4 source:', newValue)
          
          // First validate that the source can be resolved
          const isValid = await this.validateAndUpdateIPv4Config(newValue)
          if (isValid) {
            // Refresh to get the updated resolved IP from the new configuration
            this.refreshIPv4()
          }
        }, 500) // 500ms debounce
      }
    },

    // Validate and update IPv6 configuration when source address changes (debounced)
    'dnsConfig.ipv6.source'(newValue, oldValue) {
      if (newValue !== oldValue && newValue.trim() && this.initialLoadingComplete) {
        console.log('IPv6 source changed:', newValue)
        
        // Clear existing timeout
        if (this.ipv6Timeout) {
          clearTimeout(this.ipv6Timeout)
        }
        
        // Set new timeout for debounced validation and configuration update
        this.ipv6Timeout = setTimeout(async () => {
          console.log('Validating IPv6 source:', newValue)
          
          // First validate that the source can be resolved
          const isValid = await this.validateAndUpdateIPv6Config(newValue)
          if (isValid) {
            // Refresh to get the updated resolved IP from the new configuration
            this.refreshIPv6()
          }
        }, 500) // 500ms debounce
      }
    }
  },
  methods: {
    // Initial external IP detection methods
    async getInitialExternalIPv4() {
      try {
        const data = await apiGet('dns/external/ipv4');
        
        if (data) {
          this.dnsConfig.ipv4.source = data.source || '';
          this.dnsConfig.ipv4.current = data.address || '';
          this.dnsConfig.ipv4.method = data.method || 'dns';
          this.dnsConfig.ipv4.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : null;
          
          if (data.error) {
            this.dnsConfig.ipv4.current = `Error: ${data.error}`;
          } else if (data.source && data.address) {
            // Store as last valid configuration if no error
            this.lastValidIPv4Source = data.source;
          }
          
          console.log('Initial external IPv4 loaded:', {
            source: data.source,
            resolved: data.address,
            method: data.method
          });
        }
      } catch (error) {
        console.error('Error fetching initial external IPv4:', error);
      }
    },

    async getInitialExternalIPv6() {
      try {
        const data = await apiGet('dns/external/ipv6');
        
        if (data) {
          this.dnsConfig.ipv6.source = data.source || '';
          this.dnsConfig.ipv6.current = data.address || '';
          this.dnsConfig.ipv6.method = data.method || 'dns';
          this.dnsConfig.ipv6.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : null;
          
          if (data.error) {
            this.dnsConfig.ipv6.current = `Error: ${data.error}`;
          } else if (data.source && data.address) {
            // Store as last valid configuration if no error
            this.lastValidIPv6Source = data.source;
          }
          
          console.log('Initial external IPv6 loaded:', {
            source: data.source,
            resolved: data.address,
            method: data.method
          });
        }
      } catch (error) {
        console.error('Error fetching initial external IPv6:', error);
      }
    },

    // Validation and configuration update methods
    async validateAndUpdateIPv4Config(sourceAddress) {
      try {
        // First validate that the source address can be resolved
        const data = await apiGet(`dns/ipv4?hostname=${encodeURIComponent(sourceAddress)}`);
        
        if (data && data.ip) {
          console.log('IPv4 source validation successful:', data.ip);
          
          // DNS resolution succeeded, update the configuration
          await this.updateExternalIPv4Config(sourceAddress);
          
          // Store as last valid configuration
          this.lastValidIPv4Source = sourceAddress;
          return true;
        } else {
          console.warn('IPv4 source validation failed:', data.error);
          
          // Update current field to show the error
          this.dnsConfig.ipv4.current = `Error: ${data.error || 'DNS resolution failed'}`;
          this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString();
          return false;
        }
      } catch (error) {
        console.error('Error validating IPv4 source:', error);
        this.dnsConfig.ipv4.current = 'Network Error';
        this.dnsConfig.ipv4.lastUpdate = new Date().toLocaleString();
        return false;
      }
    },

    async validateAndUpdateIPv6Config(sourceAddress) {
      try {
        // First validate that the source address can be resolved
        const data = await apiGet(`dns/ipv6?hostname=${encodeURIComponent(sourceAddress)}`);
        
        if (data && data.ip) {
          console.log('IPv6 source validation successful:', data.ip);
          
          // DNS resolution succeeded, update the configuration
          await this.updateExternalIPv6Config(sourceAddress);
          
          // Store as last valid configuration
          this.lastValidIPv6Source = sourceAddress;
          return true;
        } else {
          console.warn('IPv6 source validation failed:', data.error);
          
          // Update current field to show the error
          this.dnsConfig.ipv6.current = `Error: ${data.error || 'DNS resolution failed'}`;
          this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString();
          return false;
        }
      } catch (error) {
        console.error('Error validating IPv6 source:', error);
        this.dnsConfig.ipv6.current = 'Network Error';
        this.dnsConfig.ipv6.lastUpdate = new Date().toLocaleString();
        return false;
      }
    },

    // Configuration update methods
    async updateExternalIPv4Config(sourceAddress) {
      try {
        await apiPost('dns/external/ipv4', {
          method: 'dns',
          source: sourceAddress
        });

        console.log('External IPv4 configuration updated:', sourceAddress);
      } catch (error) {
        console.error('Error updating external IPv4 config:', error);
      }
    },

    async updateExternalIPv6Config(sourceAddress) {
      try {
        await apiPost('dns/external/ipv6', {
          method: 'dns',
          source: sourceAddress
        });

        console.log('External IPv6 configuration updated:', sourceAddress);
      } catch (error) {
        console.error('Error updating external IPv6 config:', error);
      }
    },

    // Revert methods for invalid configurations
    async revertIPv4Source() {
      if (this.lastValidIPv4Source) {
        console.log('Reverting IPv4 source to:', this.lastValidIPv4Source);
        this.dnsConfig.ipv4.source = this.lastValidIPv4Source;
        // The watcher will handle validation and refresh
      }
    },

    async revertIPv6Source() {
      if (this.lastValidIPv6Source) {
        console.log('Reverting IPv6 source to:', this.lastValidIPv6Source);
        this.dnsConfig.ipv6.source = this.lastValidIPv6Source;
        // The watcher will handle validation and refresh
      }
    },

    // IP Address refresh methods
    async refreshIPv4() {
      this.ipv4Loading = true;
      try {
        const data = await apiGet('dns/external/ipv4');
        
        if (data) {
          this.dnsConfig.ipv4.current = data.address || '';
          this.dnsConfig.ipv4.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : new Date().toLocaleString();
          
          if (data.error) {
            this.dnsConfig.ipv4.current = `Error: ${data.error}`;
          }
          
          console.log('IPv4 refreshed:', this.dnsConfig.ipv4.current);
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
      this.ipv6Loading = true;
      try {
        const data = await apiGet('dns/external/ipv6');
        
        if (data) {
          this.dnsConfig.ipv6.current = data.address || '';
          this.dnsConfig.ipv6.lastUpdate = data.timestamp ? new Date(data.timestamp).toLocaleString() : new Date().toLocaleString();
          
          if (data.error) {
            this.dnsConfig.ipv6.current = `Error: ${data.error}`;
          }
          
          console.log('IPv6 refreshed:', this.dnsConfig.ipv6.current);
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
        console.log('Loading DNS configuration from existing endpoints...');
        
        // Load current IP detection settings from existing DNS endpoints
        // Since we don't have a config endpoint, we'll use the current IP detection results
        // and let users configure via the DNS tab
        
        // Auto-refresh current IPs 
        const refreshPromises = [];
        refreshPromises.push(this.refreshIPv4());
        refreshPromises.push(this.refreshIPv6());
        
        await Promise.all(refreshPromises);
        
        console.log('DNS configuration initialized with current IP detection');
      } catch (error) {
        console.error('Failed to initialize DNS configuration:', error);
        console.log('Using default DNS configuration');
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
          }
        };
        
        await apiPost('config', configData);
        
        console.log('Configuration saved successfully');
        // TODO: Show success message to user
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