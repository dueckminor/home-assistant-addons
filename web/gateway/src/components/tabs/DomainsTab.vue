<template>
  <!-- Managed Domains -->
  <v-row>
    <v-col cols="12">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="purple">mdi-sitemap</v-icon>
          Domains & Routes Configuration
        </v-card-title>
        <v-card-subtitle>
          Add domains and configure their routes
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
          <!-- Domain Tree List -->
          <v-row v-if="!loading && domains && domains.length > 0" class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <div class="d-flex justify-space-between align-center mb-3">
                <h4 class="text-subtitle-1">
                  <v-icon class="me-2" size="small">mdi-sitemap</v-icon>
                  Configured Domains & Routes ({{ domains.length }} domains)
                </h4>
                <div class="d-flex align-center">
                  <v-btn
                    icon="mdi-refresh"
                    variant="outlined"
                    size="x-small"
                    @click="loadDomains"
                    :loading="loading"
                    title="Refresh domains"
                    class="me-3"
                  ></v-btn>
                  <v-btn
                    color="purple"
                    variant="elevated"
                    prepend-icon="mdi-web-plus"
                    @click="openAddDomainWizard"
                    size="small"
                  >
                    Add Domain
                  </v-btn>
                </div>
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
                      <v-icon class="me-3" size="small">mdi-web</v-icon>
                      <!-- Domain Name -->
                      <div class="flex-grow-1">
                        <div class="text-subtitle-2">{{ domain.name }}</div>
                        <!-- Target Gateway (for redirect domains) -->
                        <div v-if="domain.redirect" class="text-caption text-medium-emphasis">
                          → {{ domain.redirect.target }}
                        </div>
                      </div>
                      
                      <!-- Status Indicators -->
                      <div class="d-flex align-center">
                        <!-- DNS Status (only for regular domains) -->
                        <v-tooltip v-if="!domain.redirect" :text="getDnsStatusTooltip(domain)">
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
                        
                        <!-- Certificate Status (only for regular domains) -->
                        <v-tooltip v-if="!domain.redirect" :text="getCertificateTooltip(domain)">
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
                        
                        <!-- Domain Type Chip -->
                        <v-chip
                          v-if="domain.redirect"
                          color="orange"
                          size="x-small"
                          class="me-2"
                        >
                          <v-icon start size="x-small">mdi-redirect</v-icon>
                          Redirect
                        </v-chip>
                        
                        <!-- Routes Count (only for non-redirect domains) -->
                        <v-chip
                          v-else
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
                    <!-- Redirect Configuration (for redirect domains) -->
                    <div v-if="domain.redirect" class="mb-4">
                      <v-card variant="tonal" color="orange">
                        <v-card-text class="pa-3">
                          <div class="d-flex align-center mb-3">
                            <v-icon class="me-2" size="small">mdi-redirect</v-icon>
                            <span class="text-subtitle-2">Gateway Redirect Configuration</span>
                          </div>
                          
                          <v-row>
                            <v-col cols="12" sm="6" md="3">
                              <div class="text-caption text-medium-emphasis mb-1">Target Gateway</div>
                              <div class="text-body-2 font-weight-medium">{{ domain.redirect.target }}</div>
                            </v-col>
                            <v-col cols="12" sm="6" md="3">
                              <div class="text-caption text-medium-emphasis mb-1">HTTP Port</div>
                              <div class="text-body-2 font-weight-medium">{{ domain.redirect.http_port }}</div>
                            </v-col>
                            <v-col cols="12" sm="6" md="3">
                              <div class="text-caption text-medium-emphasis mb-1">HTTPS Port</div>
                              <div class="text-body-2 font-weight-medium">{{ domain.redirect.https_port }}</div>
                            </v-col>
                            <v-col cols="12" sm="6" md="3">
                              <div class="text-caption text-medium-emphasis mb-1">DNS Port</div>
                              <div class="text-body-2 font-weight-medium">{{ domain.redirect.dns_port }}</div>
                            </v-col>
                          </v-row>
                        </v-card-text>
                      </v-card>
                    </div>

                    <!-- Domain Status Details (for regular domains only) -->
                    <div v-else class="mb-4">
                      <v-row>
                        <v-col cols="12" md="6">
                          <v-card variant="tonal" color="info">
                            <v-card-text class="pa-3">
                              <div class="d-flex align-center mb-2">
                                <v-icon class="me-2" size="small">mdi-dns</v-icon>
                                <span class="text-subtitle-2">DNS Configuration</span>
                                <v-spacer></v-spacer>
                                <v-btn
                                  icon="mdi-refresh"
                                  size="x-small"
                                  variant="text"
                                  @click="checkDomainDnsOnly(domain)"
                                  :loading="domain.dnsChecking"
                                  title="Refresh DNS status"
                                ></v-btn>
                                <v-btn
                                  v-if="domain.dnsStatus"
                                  :icon="domain.showDnsDetails ? 'mdi-chevron-up' : 'mdi-chevron-down'"
                                  size="x-small"
                                  variant="text"
                                  @click="domain.showDnsDetails = !domain.showDnsDetails"
                                  :title="domain.showDnsDetails ? 'Hide DNS details' : 'Show DNS details'"
                                ></v-btn>
                              </div>
                              <div class="text-body-2">
                                Status: <v-chip :color="getDnsStatusColor(domain)" size="x-small">{{ getDnsStatusText(domain) }}</v-chip>
                              </div>
                              <div v-if="domain.dnsStatus?.hostnameChecked" class="text-caption mt-1 text-medium-emphasis">
                                Checking: {{ domain.dnsStatus.hostnameChecked }}
                              </div>
                              
                              <!-- Expandable DNS Details -->
                              <v-expand-transition>
                                <div v-if="domain.showDnsDetails && domain.dnsStatus" class="mt-3 pt-2 border-t-thin">
                                  <div class="text-caption">
                                    <!-- IPv4 Status -->
                                    <div v-if="domain.dnsStatus.ipv4" class="mb-2">
                                      <div class="d-flex align-center mb-1">
                                        <v-icon 
                                          size="x-small" 
                                          :color="domain.dnsStatus.ipv4.status === 'ok' ? 'success' : domain.dnsStatus.ipv4.status === 'warning' ? 'warning' : 'error'"
                                          class="me-1"
                                        >
                                          {{ domain.dnsStatus.ipv4.status === 'ok' ? 'mdi-check' : domain.dnsStatus.ipv4.status === 'warning' ? 'mdi-alert' : 'mdi-close' }}
                                        </v-icon>
                                        <strong>IPv4 (A Record)</strong>
                                      </div>
                                      <div class="ms-4">
                                        <div>Expected: {{ domain.dnsStatus.ipv4.expected }}</div>
                                        <div v-if="domain.dnsStatus.ipv4.actual?.length" class="text-medium-emphasis">
                                          Actual: {{ domain.dnsStatus.ipv4.actual.join(', ') }}
                                        </div>
                                        <div v-else class="text-medium-emphasis">
                                          Actual: No records found
                                        </div>
                                      </div>
                                    </div>
                                    
                                    <!-- IPv6 Status -->
                                    <div v-if="domain.dnsStatus.ipv6" class="mb-2">
                                      <div class="d-flex align-center mb-1">
                                        <v-icon 
                                          size="x-small" 
                                          :color="domain.dnsStatus.ipv6.status === 'ok' ? 'success' : domain.dnsStatus.ipv6.status === 'warning' ? 'warning' : 'error'"
                                          class="me-1"
                                        >
                                          {{ domain.dnsStatus.ipv6.status === 'ok' ? 'mdi-check' : domain.dnsStatus.ipv6.status === 'warning' ? 'mdi-alert' : 'mdi-close' }}
                                        </v-icon>
                                        <strong>IPv6 (AAAA Record)</strong>
                                      </div>
                                      <div class="ms-4">
                                        <div>Expected: {{ domain.dnsStatus.ipv6.expected }}</div>
                                        <div v-if="domain.dnsStatus.ipv6.actual?.length" class="text-medium-emphasis">
                                          Actual: {{ domain.dnsStatus.ipv6.actual.join(', ') }}
                                        </div>
                                        <div v-else class="text-medium-emphasis">
                                          Actual: No records found
                                        </div>
                                      </div>
                                    </div>
                                    
                                    <!-- Last Checked -->
                                    <div class="text-medium-emphasis">
                                      Last checked: {{ formatDate(domain.dnsStatus.lastChecked) }}
                                    </div>
                                  </div>
                                </div>
                              </v-expand-transition>

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
                              <div v-if="domain.server_certificate" class="text-caption mt-1 text-medium-emphasis">
                                <div v-if="domain.server_certificate.valid_not_before">
                                  Valid from: {{ formatDate(domain.server_certificate.valid_not_before) }}
                                </div>
                                <div v-if="domain.server_certificate.valid_not_after">
                                  Valid until: {{ formatDate(domain.server_certificate.valid_not_after) }}
                                </div>
                              </div>
                            </v-card-text>
                          </v-card>
                        </v-col>
                      </v-row>
                    </div>
                    
                    <!-- Routes Section (only for regular domains, not redirects) -->
                    <div v-if="!domain.redirect">
                      <div class="d-flex justify-space-between align-center mb-3">
                        <h5 class="text-subtitle-2">
                          <v-icon class="me-2" size="small">mdi-routes</v-icon>
                          Routes ({{ (domain.routes || []).length }})
                        </h5>
                        <v-btn
                          size="small"
                          color="primary"
                          prepend-icon="mdi-plus"
                          @click="openAddRouteWizard(domain.guid)"
                        >
                          Add Route
                        </v-btn>
                      </div>
                      
                      <!-- Routes List -->
                      <div v-if="domain.routes && domain.routes.length > 0">
                        <v-card
                          v-for="route in domain.routes"
                          :key="route.guid"
                          variant="outlined"
                          class="mb-2 route-card"
                        >
                          <v-card-text class="pa-3">
                            <div class="d-flex align-center">
                              <v-icon class="me-3" size="small" color="primary">mdi-dns</v-icon>
                              <div class="flex-grow-1">
                                <div class="text-body-2 font-weight-medium">{{ route.hostname }}</div>
                                <div class="text-caption text-medium-emphasis">
                                  → {{ route.target || 'No target configured' }}
                                </div>
                              </div>
                              <v-btn
                                icon="mdi-pencil"
                                variant="text"
                                size="small"
                                @click="openEditRouteWizard(domain.guid, route)"
                                title="Edit route"
                              ></v-btn>
                              <v-btn
                                icon="mdi-delete"
                                variant="text"
                                size="small"
                                color="error"
                                @click="removeRoute(domain.guid, route.guid)"
                                title="Delete route"
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
                            @click="openAddRouteWizard(domain.guid)"
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

  <!-- Route Wizard -->
  <RouteWizard
    v-model="showRouteWizard"
    :domain-guid="selectedDomainGuid"
    :domain-name="selectedDomainName"
    :edit-route="editingRoute"
    @route-saved="onRouteSaved"
  />

  <!-- Domain Wizard -->
  <DomainWizard
    v-model="showDomainWizard"
    @domain-saved="onDomainSaved"
  />
</template>

<script>
import { apiGet, apiRequest } from '../../../../shared/utils/homeassistant.js'
import RouteWizard from '../dialogs/RouteWizard.vue'
import DomainWizard from '../dialogs/DomainWizard.vue'

export default {
  name: 'DomainsTab',
  components: {
    RouteWizard,
    DomainWizard
  },
  data() {
    return {
      domains: [],
      loading: false,
      error: null,
      showRouteWizard: false,
      showDomainWizard: false,
      selectedDomainGuid: null,
      selectedDomainName: null,
      editingRoute: null
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
        
        // Initialize UI state for each domain
        this.domains.forEach(domain => {
          domain.showDnsDetails = false
        })
        
        // Check DNS status for each domain asynchronously (don't await)
        this.checkDnsStatusForAllDomains().catch(err => {
          console.error('Error checking DNS status:', err)
        })
      } catch (err) {
        this.error = `Failed to load domains: ${err.message}`
        console.error('Error loading domains:', err)
      } finally {
        this.loading = false
      }
    },

    // DNS Status Checking Methods
    async checkDnsStatusForAllDomains() {
      // Get external IPs once
      const [externalIpv4, externalIpv6] = await Promise.allSettled([
        this.getExternalIp('ipv4'),
        this.getExternalIp('ipv6')
      ])

      // Check DNS for each domain
      for (const domain of this.domains) {
        if (!domain.redirect) { // Only check DNS for non-redirect domains
          domain.dnsStatus = await this.checkDomainDnsStatus(domain, externalIpv4, externalIpv6)
        }
      }
    },

    async getExternalIp(version) {
      try {
        const response = await apiGet(`dns/external/${version}`)
        return response.address
      } catch (error) {
        console.warn(`Failed to get external ${version}:`, error)
        return null
      }
    },

    async checkDomainDnsStatus(domain, externalIpv4Result, externalIpv6Result) {
      const status = {
        status: 'ok',
        ipv4: null,
        ipv6: null,
        hostnameChecked: null,
        lastChecked: new Date().toISOString()
      }

      const expectedIpv4 = externalIpv4Result.status === 'fulfilled' ? externalIpv4Result.value : null
      const expectedIpv6 = externalIpv6Result.status === 'fulfilled' ? externalIpv6Result.value : null

      // Determine hostname to check - use wildcard or first route hostname
      let hostnameToCheck = `*.${domain.name}` // Default to wildcard
      
      // If domain has routes, check the first route's hostname instead
      if (domain.routes && domain.routes.length > 0) {
        hostnameToCheck = `${domain.routes[0].hostname}.${domain.name}`
      }

      // Store which hostname we're checking
      status.hostnameChecked = hostnameToCheck

      // Check IPv4 (A record)
      if (expectedIpv4) {
        status.ipv4 = await this.checkDnsRecord(hostnameToCheck, 'A', expectedIpv4)
      }

      // Check IPv6 (AAAA record)
      if (expectedIpv6) {
        status.ipv6 = await this.checkDnsRecord(hostnameToCheck, 'AAAA', expectedIpv6)
      }

      // Determine overall status
      const ipv4Status = status.ipv4?.status || 'unknown'
      const ipv6Status = status.ipv6?.status || 'unknown'

      if (ipv4Status === 'error' || ipv6Status === 'error') {
        status.status = 'error'
      } else if (ipv4Status === 'warning' || ipv6Status === 'warning') {
        status.status = 'warning'
      } else if (ipv4Status === 'ok' || ipv6Status === 'ok') {
        status.status = 'ok'
      } else {
        status.status = 'unknown'
      }

      return status
    },

    async checkDomainDnsOnly(domain) {
      if (domain.redirect) return // Skip redirect domains
      
      try {
        domain.dnsChecking = true
        
        // Get external IPs
        const [externalIpv4, externalIpv6] = await Promise.allSettled([
          this.getExternalIp('ipv4'),
          this.getExternalIp('ipv6')
        ])
        
        // Update DNS status for this domain
        domain.dnsStatus = await this.checkDomainDnsStatus(domain, externalIpv4, externalIpv6)
      } catch (error) {
        console.error('Error checking DNS for domain:', domain.name, error)
      } finally {
        domain.dnsChecking = false
      }
    },

    async checkDnsRecord(domainName, recordType, expectedIp) {
      try {
        const response = await apiGet(`dns/lookup?hostname=${domainName}&type=${recordType}`)
        
        if (response.records && response.records.length > 0) {
          // Check if any of the returned records match the expected IP
          const actualIps = response.records.map(record => record.value || record.ip || record)
          const matches = actualIps.includes(expectedIp)
          
          return {
            status: matches ? 'ok' : 'error',
            expected: expectedIp,
            actual: actualIps,
            matches
          }
        } else {
          return {
            status: 'error',
            expected: expectedIp,
            actual: [],
            error: 'No records found'
          }
        }
      } catch (error) {
        // Handle 404 as "no records found"
        if (error.message.includes('404')) {
          return {
            status: 'warning',
            expected: expectedIp,
            actual: [],
            error: 'No DNS records configured'
          }
        }
        
        return {
          status: 'error',
          expected: expectedIp,
          actual: [],
          error: error.message
        }
      }
    },

    openAddDomainWizard() {
      this.showDomainWizard = true
    },

    onDomainSaved(newDomain) {
      // Add domain to local list
      this.domains.push(newDomain)
      this.error = null
      console.log('Domain added successfully:', newDomain)
    },

    async removeDomain(domainGuid) {
      try {
        console.log('Removing domain via API:', domainGuid)
        
        // Make API call to remove domain using GUID
        const response = await apiRequest(`domains/${domainGuid}`, {
          method: 'DELETE'
        })
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.status} ${response.statusText}`)
        }
        
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
      if (!domain.dnsStatus) return 'Checking...'
      
      const ipv4Status = domain.dnsStatus.ipv4?.status
      const ipv6Status = domain.dnsStatus.ipv6?.status
      
      // Count configured records
      let configuredCount = 0
      let totalRecords = 0
      
      if (domain.dnsStatus.ipv4) {
        totalRecords++
        if (ipv4Status === 'ok') configuredCount++
      }
      
      if (domain.dnsStatus.ipv6) {
        totalRecords++
        if (ipv6Status === 'ok') configuredCount++
      }
      
      if (totalRecords === 0) return 'No checks'
      if (configuredCount === totalRecords) return 'All OK'
      if (configuredCount > 0) return `${configuredCount}/${totalRecords} OK`
      
      switch (domain.dnsStatus.status) {
        case 'warning': return 'Missing'
        case 'error': return 'Error'
        default: return 'Unknown'
      }
    },

    getDnsStatusTooltip(domain) {
      if (!domain.dnsStatus) return 'DNS status not checked'
      
      const lines = []
      
      if (domain.dnsStatus.hostnameChecked) {
        lines.push(`Checking DNS for: ${domain.dnsStatus.hostnameChecked}`)
        lines.push('')
      }
      
      if (domain.dnsStatus.ipv4) {
        const ipv4 = domain.dnsStatus.ipv4
        const status = ipv4.status === 'ok' ? '✅' : ipv4.status === 'warning' ? '⚠️' : '❌'
        lines.push(`${status} IPv4 (A): Expected ${ipv4.expected}`)
        if (ipv4.actual && ipv4.actual.length > 0) {
          lines.push(`   Actual: ${ipv4.actual.join(', ')}`)
        } else {
          lines.push(`   Actual: No records found`)
        }
      }
      
      if (domain.dnsStatus.ipv6) {
        const ipv6 = domain.dnsStatus.ipv6
        const status = ipv6.status === 'ok' ? '✅' : ipv6.status === 'warning' ? '⚠️' : '❌'
        lines.push(`${status} IPv6 (AAAA): Expected ${ipv6.expected}`)
        if (ipv6.actual && ipv6.actual.length > 0) {
          lines.push(`   Actual: ${ipv6.actual.join(', ')}`)
        } else {
          lines.push(`   Actual: No records found`)
        }
      }
      
      if (lines.length === 0) {
        return 'No DNS checks performed'
      }
      
      lines.push('', `Last checked: ${new Date(domain.dnsStatus.lastChecked).toLocaleString()}`)
      return lines.join('\n')
    },

    // Certificate Status Methods
    getCertificateStatusColor(domain) {
      if (!domain.server_certificate) return 'grey'
      const cert = domain.server_certificate
      if (!cert.valid_not_after) return 'grey'
      
      // Check expiration
      const expiresAt = new Date(cert.valid_not_after)
      const now = new Date()
      const daysUntilExpiry = (expiresAt - now) / (1000 * 60 * 60 * 24)
      
      if (daysUntilExpiry < 0) return 'error' // Expired
      if (daysUntilExpiry < 7) return 'error'
      if (daysUntilExpiry < 30) return 'warning'
      return 'success'
    },

    getCertificateStatusIcon(domain) {
      if (!domain.server_certificate) return 'mdi-help'
      const cert = domain.server_certificate
      if (!cert.valid_not_after) return 'mdi-help'
      
      const expiresAt = new Date(cert.valid_not_after)
      const now = new Date()
      const daysUntilExpiry = (expiresAt - now) / (1000 * 60 * 60 * 24)
      
      if (daysUntilExpiry < 0) return 'mdi-close' // Expired
      if (daysUntilExpiry < 7) return 'mdi-alert'
      if (daysUntilExpiry < 30) return 'mdi-clock-alert'
      return 'mdi-check'
    },

    getCertificateStatusText(domain) {
      if (!domain.server_certificate) return 'None'
      const cert = domain.server_certificate
      if (!cert.valid_not_after) return 'Unknown'
      
      const expiresAt = new Date(cert.valid_not_after)
      const now = new Date()
      const daysUntilExpiry = Math.floor((expiresAt - now) / (1000 * 60 * 60 * 24))
      
      if (daysUntilExpiry < 0) return 'Expired'
      if (daysUntilExpiry < 7) return `Expires in ${daysUntilExpiry}d`
      if (daysUntilExpiry < 30) return `Expires in ${daysUntilExpiry}d`
      return 'Valid'
    },

    getCertificateTooltip(domain) {
      if (!domain.server_certificate) return 'No certificate configured'
      const cert = domain.server_certificate
      if (!cert.valid_not_after) return 'Certificate information unavailable'
      
      const validFrom = new Date(cert.valid_not_before)
      const validUntil = new Date(cert.valid_not_after)
      return `Certificate valid from ${validFrom.toLocaleDateString()} until ${validUntil.toLocaleDateString()}`
    },

    // Route Management Methods
    openAddRouteWizard(domainGuid) {
      const domain = this.domains.find(d => d.guid === domainGuid)
      this.selectedDomainGuid = domainGuid
      this.selectedDomainName = domain ? domain.name : ''
      this.editingRoute = null
      this.showRouteWizard = true
    },

    openEditRouteWizard(domainGuid, route) {
      const domain = this.domains.find(d => d.guid === domainGuid)
      this.selectedDomainGuid = domainGuid
      this.selectedDomainName = domain ? domain.name : ''
      this.editingRoute = route
      this.showRouteWizard = true
    },

    onRouteSaved(savedRoute) {
      // Update local state with the saved route
      const domain = this.domains.find(d => d.guid === this.selectedDomainGuid)
      if (domain) {
        if (this.editingRoute) {
          // Update existing route
          const routeIndex = domain.routes.findIndex(r => r.guid === this.editingRoute.guid)
          if (routeIndex !== -1) {
            domain.routes[routeIndex] = savedRoute
          }
        } else {
          // Add new route
          if (!domain.routes) domain.routes = []
          domain.routes.push(savedRoute)
        }
      }
      
      // Clear wizard state
      this.selectedDomainGuid = null
      this.editingRoute = null
    },

    async removeRoute(domainGuid, routeGuid) {
      try {
        console.log('Remove route:', routeGuid, 'from domain:', domainGuid)
        
        // Call the route deletion API
        await apiRequest(`domains/${domainGuid}/routes/${routeGuid}`, {
          method: 'DELETE'
        })
        
        // Update local state - remove the route from the domain
        const domain = this.domains.find(d => d.guid === domainGuid)
        if (domain && domain.routes) {
          domain.routes = domain.routes.filter(r => r.guid !== routeGuid)
        }
        
        console.log('Route removed successfully')
      } catch (error) {
        this.error = `Failed to remove route: ${error.message}`
        console.error('Error removing route:', error)
      }
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