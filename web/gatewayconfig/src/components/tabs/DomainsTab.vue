<template>
  <v-row>
    <v-col cols="12">
      <div class="text-center mb-6">
        <v-icon size="48" color="purple" class="mb-3">mdi-sitemap</v-icon>
        <h2 class="text-h5 mb-2">Domains & Routes</h2>
        <p class="text-body-2 text-medium-emphasis">
          Manage domains, routes, DNS configuration, and SSL certificates
        </p>
      </div>
    </v-col>
  </v-row>

  <!-- Managed Domains -->
  <v-row>
    <v-col cols="12">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="purple">mdi-sitemap</v-icon>
          Domains & Routes Configuration
        </v-card-title>
        <v-card-subtitle>
          Add domains and configure their routes, DNS settings, and SSL certificates
        </v-card-subtitle>
        <v-card-text>
          <!-- Error Display -->
          <v-alert
            v-if="error"
            type="error"
            variant="tonal"
            class="mb-4"
            closable
            @click:close="error = null"
          >
            {{ error }}
          </v-alert>
          <v-row>
            <v-col cols="12" md="8">
              <v-text-field
                v-model="newDomain"
                label="Add Domain"
                variant="outlined"
                prepend-inner-icon="mdi-web"
                placeholder="example.com"
                hint="Enter a domain that should be handled by this DNS server"
                persistent-hint
                @keyup.enter="addDomain"
              >
                <template v-slot:append-inner>
                  <v-btn
                    icon="mdi-plus"
                    variant="text"
                    size="small"
                    @click="addDomain"
                    :disabled="!newDomain.trim() || addingDomain"
                    :loading="addingDomain"
                  ></v-btn>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="4">
              <v-btn
                color="primary"
                variant="outlined"
                prepend-icon="mdi-plus"
                @click="addDomain"
                :disabled="!newDomain.trim() || addingDomain"
                :loading="addingDomain"
                block
              >
                Add Domain
              </v-btn>
            </v-col>
          </v-row>

          <!-- Domain Tree List -->
          <v-row v-if="!loading && domains && domains.length > 0" class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <div class="d-flex justify-space-between align-center mb-3">
                <h4 class="text-subtitle-1">
                  <v-icon class="me-2" size="small">mdi-sitemap</v-icon>
                  Configured Domains & Routes ({{ domains.length }} domains)
                </h4>
                <v-btn
                  icon="mdi-refresh"
                  variant="text"
                  size="small"
                  @click="loadDomains"
                  :loading="loading"
                  title="Refresh domains"
                ></v-btn>
              </div>
              
              <!-- Domain Tree -->
              <v-expansion-panels variant="accordion" multiple>
                <v-expansion-panel
                  v-for="domain in domains"
                  :key="domain.guid"
                  class="domain-panel"
                >
                  <v-expansion-panel-title class="domain-header">
                    <div class="d-flex align-center w-100">
                      <!-- Domain Name -->
                      <div class="flex-grow-1">
                        <div class="d-flex align-center">
                          <v-icon class="me-2" size="small">mdi-web</v-icon>
                          <span class="text-subtitle-2">{{ domain.name }}</span>
                        </div>
                      </div>
                      
                      <!-- Status Indicators -->
                      <div class="d-flex align-center me-4">
                        <!-- DNS Status -->
                        <v-tooltip text="DNS Configuration Status">
                          <template v-slot:activator="{ props }">
                            <v-chip
                              v-bind="props"
                              :color="getDnsStatusColor(domain)"
                              size="x-small"
                              class="me-2"
                            >
                              <v-icon start size="x-small">{{ getDnsStatusIcon(domain) }}</v-icon>
                              DNS
                            </v-chip>
                          </template>
                        </v-tooltip>
                        
                        <!-- Certificate Status -->
                        <v-tooltip :text="getCertificateTooltip(domain)">
                          <template v-slot:activator="{ props }">
                            <v-chip
                              v-bind="props"
                              :color="getCertificateStatusColor(domain)"
                              size="x-small"
                              class="me-2"
                            >
                              <v-icon start size="x-small">{{ getCertificateStatusIcon(domain) }}</v-icon>
                              SSL
                            </v-chip>
                          </template>
                        </v-tooltip>
                        
                        <!-- Routes Count -->
                        <v-chip
                          color="info"
                          size="x-small"
                          class="me-2"
                        >
                          <v-icon start size="x-small">mdi-routes</v-icon>
                          {{ (domain.routes || []).length }}
                        </v-chip>
                        
                        <!-- Delete Button -->
                        <v-btn
                          icon="mdi-delete"
                          variant="text"
                          size="small"
                          color="error"
                          @click.stop="removeDomain(domain.guid)"
                          :title="`Remove ${domain.name}`"
                        ></v-btn>
                      </div>
                    </div>
                  </v-expansion-panel-title>
                  
                  <v-expansion-panel-text>
                    <!-- Domain Status Details -->
                    <div class="mb-4">
                      <v-row>
                        <v-col cols="12" md="6">
                          <v-card variant="tonal" color="info">
                            <v-card-text class="pa-3">
                              <div class="d-flex align-center mb-2">
                                <v-icon class="me-2" size="small">mdi-dns</v-icon>
                                <span class="text-subtitle-2">DNS Configuration</span>
                              </div>
                              <div class="text-body-2">
                                Status: <v-chip :color="getDnsStatusColor(domain)" size="x-small">{{ getDnsStatusText(domain) }}</v-chip>
                              </div>
                              <div class="text-caption mt-1 text-medium-emphasis">
                                Last checked: {{ formatDate(domain.lastDnsCheck) }}
                              </div>
                            </v-card-text>
                          </v-card>
                        </v-col>
                        <v-col cols="12" md="6">
                          <v-card variant="tonal" color="success">
                            <v-card-text class="pa-3">
                              <div class="d-flex align-center mb-2">
                                <v-icon class="me-2" size="small">mdi-certificate</v-icon>
                                <span class="text-subtitle-2">SSL Certificate</span>
                              </div>
                              <div class="text-body-2">
                                Status: <v-chip :color="getCertificateStatusColor(domain)" size="x-small">{{ getCertificateStatusText(domain) }}</v-chip>
                              </div>
                              <div class="text-caption mt-1 text-medium-emphasis" v-if="domain.certificate && domain.certificate.expiresAt">
                                Expires: {{ formatDate(domain.certificate.expiresAt) }}
                              </div>
                            </v-card-text>
                          </v-card>
                        </v-col>
                      </v-row>
                    </div>
                    
                    <!-- Routes Section -->
                    <div>
                      <div class="d-flex justify-space-between align-center mb-3">
                        <h5 class="text-subtitle-2">
                          <v-icon class="me-2" size="small">mdi-routes</v-icon>
                          Routes ({{ (domain.routes || []).length }})
                        </h5>
                        <v-btn
                          size="small"
                          color="primary"
                          prepend-icon="mdi-plus"
                          @click="addRoute(domain.guid)"
                        >
                          Add Route
                        </v-btn>
                      </div>
                      
                      <!-- Routes List -->
                      <div v-if="domain.routes && domain.routes.length > 0">
                        <v-card
                          v-for="route in domain.routes"
                          :key="route.guid || route.path"
                          variant="outlined"
                          class="mb-2 route-card"
                        >
                          <v-card-text class="pa-3">
                            <div class="d-flex align-center">
                              <v-icon class="me-3" size="small" color="primary">mdi-arrow-right</v-icon>
                              <div class="flex-grow-1">
                                <div class="text-body-2 font-weight-medium">{{ route.path || '/' }}</div>
                                <div class="text-caption text-medium-emphasis">
                                  → {{ route.target || 'No target configured' }}
                                </div>
                              </div>
                              <v-chip
                                :color="route.enabled ? 'success' : 'warning'"
                                size="x-small"
                                class="me-2"
                              >
                                {{ route.enabled ? 'Active' : 'Inactive' }}
                              </v-chip>
                              <v-btn
                                icon="mdi-pencil"
                                variant="text"
                                size="small"
                                @click="editRoute(domain.guid, route.guid || route.path)"
                              ></v-btn>
                              <v-btn
                                icon="mdi-delete"
                                variant="text"
                                size="small"
                                color="error"
                                @click="removeRoute(domain.guid, route.guid || route.path)"
                              ></v-btn>
                            </div>
                          </v-card-text>
                        </v-card>
                      </div>
                      
                      <!-- Empty Routes State -->
                      <v-card v-else variant="tonal" color="grey">
                        <v-card-text class="text-center pa-4">
                          <v-icon size="32" color="grey" class="mb-2">mdi-routes-clock</v-icon>
                          <p class="text-body-2 text-medium-emphasis mb-2">
                            No routes configured for this domain
                          </p>
                          <v-btn
                            size="small"
                            color="primary"
                            prepend-icon="mdi-plus"
                            @click="addRoute(domain.guid)"
                          >
                            Add First Route
                          </v-btn>
                        </v-card-text>
                      </v-card>
                    </div>
                  </v-expansion-panel-text>
                </v-expansion-panel>
              </v-expansion-panels>
            </v-col>
          </v-row>

          <!-- Loading State -->
          <v-row v-else-if="loading" class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <div class="text-center pa-4">
                <v-progress-circular
                  indeterminate
                  color="primary"
                  size="48"
                  class="mb-2"
                ></v-progress-circular>
                <p class="text-body-2 text-medium-emphasis">
                  Loading domains...
                </p>
              </div>
            </v-col>
          </v-row>

          <!-- Empty State -->
          <v-row v-else class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <div class="text-center pa-4">
                <v-icon size="48" color="grey" class="mb-2">mdi-domain-off</v-icon>
                <p class="text-body-2 text-medium-emphasis">
                  No domains configured. Add domains that should be handled by this DNS server.
                </p>
              </div>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>

  <!-- DNS Server Status -->
  <v-row class="mt-4">
    <v-col cols="12">
      <v-card color="grey-lighten-5">
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="success">mdi-server</v-icon>
          DNS Server Status
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="3">
              <v-chip color="success" class="ma-1">
                <v-icon start>mdi-port</v-icon>
                Port: 53 (UDP/TCP)
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="info" class="ma-1">
                <v-icon start>mdi-earth</v-icon>
                External IP Detection
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="purple" class="ma-1">
                <v-icon start>mdi-domain</v-icon>
                {{ (domains && domains.length) || 0 }} Domains
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="primary" class="ma-1">
                <v-icon start>mdi-check-circle</v-icon>
                Running
              </v-chip>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>

<script>
import { apiRequest, apiGet, apiPost } from '../../utils/api.js'

export default {
  name: 'DomainsTab',
  data() {
    return {
      domains: [],
      newDomain: '',
      loading: false,
      addingDomain: false,
      error: null
    }
  },
  async mounted() {
    await this.loadDomains()
  },
  methods: {
    async loadDomains() {
      try {
        this.loading = true
        this.error = null
        const response = await apiGet('domains')
        this.domains = response.domains || []
      } catch (err) {
        this.error = `Failed to load domains: ${err.message}`
        console.error('Error loading domains:', err)
      } finally {
        this.loading = false
      }
    },

    async addDomain() {
      const domainName = this.newDomain.trim().toLowerCase();
      if (!domainName) return;
      
      // Basic domain validation
      if (!/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(domainName)) {
        this.error = 'Invalid domain format';
        return;
      }
      
      // Check for duplicates
      if (this.domains.some(d => d.name === domainName)) {
        this.error = 'Domain already exists';
        return;
      }
      
      this.addingDomain = true
      try {
        console.log('Adding domain via API:', domainName)
        
        // Make API call to add domain using new REST API
        const newDomain = await apiRequest('domains', 'POST', { name: domainName })
        
        // Add domain to local list
        this.domains.push(newDomain);
        this.newDomain = '';
        this.error = null;
        console.log('Domain added successfully:', newDomain);
        
      } catch (error) {
        this.error = `Failed to add domain: ${error.message}`;
        console.error('Error adding domain:', error)
      } finally {
        this.addingDomain = false
      }
    },

    async removeDomain(domainGuid) {
      try {
        console.log('Removing domain via API:', domainGuid)
        
        // Make API call to remove domain using GUID
        await apiRequest(`domains/${domainGuid}`, 'DELETE')
        
        // Remove domain from local list
        this.domains = this.domains.filter(d => d.guid !== domainGuid);
        this.error = null;
        console.log('Domain removed successfully:', domainGuid);
        
      } catch (error) {
        this.error = `Failed to remove domain: ${error.message}`;
        console.error('Error removing domain:', error)
      }
    },

    // DNS Status Methods
    getDnsStatusColor(domain) {
      if (!domain.dnsStatus) return 'grey'
      switch (domain.dnsStatus.status) {
        case 'ok': return 'success'
        case 'warning': return 'warning'
        case 'error': return 'error'
        default: return 'grey'
      }
    },

    getDnsStatusIcon(domain) {
      if (!domain.dnsStatus) return 'mdi-help'
      switch (domain.dnsStatus.status) {
        case 'ok': return 'mdi-check'
        case 'warning': return 'mdi-alert'
        case 'error': return 'mdi-close'
        default: return 'mdi-help'
      }
    },

    getDnsStatusText(domain) {
      if (!domain.dnsStatus) return 'Unknown'
      switch (domain.dnsStatus.status) {
        case 'ok': return 'Configured'
        case 'warning': return 'Warning'
        case 'error': return 'Error'
        default: return 'Unknown'
      }
    },

    // Certificate Status Methods
    getCertificateStatusColor(domain) {
      if (!domain.certificate) return 'grey'
      const cert = domain.certificate
      if (!cert.valid) return 'error'
      
      // Check expiration
      const expiresAt = new Date(cert.expiresAt)
      const now = new Date()
      const daysUntilExpiry = (expiresAt - now) / (1000 * 60 * 60 * 24)
      
      if (daysUntilExpiry < 7) return 'error'
      if (daysUntilExpiry < 30) return 'warning'
      return 'success'
    },

    getCertificateStatusIcon(domain) {
      if (!domain.certificate) return 'mdi-help'
      const cert = domain.certificate
      if (!cert.valid) return 'mdi-close'
      
      const expiresAt = new Date(cert.expiresAt)
      const now = new Date()
      const daysUntilExpiry = (expiresAt - now) / (1000 * 60 * 60 * 24)
      
      if (daysUntilExpiry < 7) return 'mdi-alert'
      if (daysUntilExpiry < 30) return 'mdi-clock-alert'
      return 'mdi-check'
    },

    getCertificateStatusText(domain) {
      if (!domain.certificate) return 'None'
      const cert = domain.certificate
      if (!cert.valid) return 'Invalid'
      
      const expiresAt = new Date(cert.expiresAt)
      const now = new Date()
      const daysUntilExpiry = Math.floor((expiresAt - now) / (1000 * 60 * 60 * 24))
      
      if (daysUntilExpiry < 0) return 'Expired'
      if (daysUntilExpiry < 7) return `Expires in ${daysUntilExpiry}d`
      if (daysUntilExpiry < 30) return `Expires in ${daysUntilExpiry}d`
      return 'Valid'
    },

    getCertificateTooltip(domain) {
      if (!domain.certificate) return 'No certificate configured'
      const cert = domain.certificate
      if (!cert.valid) return 'Certificate is invalid'
      
      const expiresAt = new Date(cert.expiresAt)
      return `Certificate expires on ${expiresAt.toLocaleDateString()}`
    },

    // Route Management Methods (placeholders for future implementation)
    addRoute(domainGuid) {
      console.log('Add route to domain:', domainGuid)
      // TODO: Implement route addition dialog/form
    },

    editRoute(domainGuid, routeGuid) {
      console.log('Edit route:', routeGuid, 'for domain:', domainGuid)
      // TODO: Implement route editing dialog/form
    },

    removeRoute(domainGuid, routeGuid) {
      console.log('Remove route:', routeGuid, 'from domain:', domainGuid)
      // TODO: Implement route removal API call
    },

    // Utility Methods
    formatDate(dateString) {
      if (!dateString) return 'Never'
      return new Date(dateString).toLocaleString()
    }
  }
}
</script>

<style scoped>
.domain-panel {
  margin-bottom: 8px !important;
}

.domain-header {
  padding: 12px 16px !important;
}

.route-card {
  transition: all 0.2s ease;
}

.route-card:hover {
  transform: translateX(4px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* Override expansion panel styles for better spacing */
:deep(.v-expansion-panel-text__wrapper) {
  padding: 16px 20px 20px 20px;
}

:deep(.v-expansion-panel-title) {
  min-height: 64px;
}

/* Status indicator styling */
.v-chip.v-chip--size-x-small {
  font-size: 0.7rem;
  height: 20px;
}
</style>