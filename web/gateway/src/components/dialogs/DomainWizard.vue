<template>
  <v-dialog
    v-model="localShow"
    max-width="600px"
    persistent
    scrollable
  >
    <v-card>
      <v-card-title class="text-h5 d-flex align-center bg-purple text-white">
        <v-icon class="me-2">mdi-web-plus</v-icon>
        Add New Domain
      </v-card-title>
      
      <v-card-text class="pa-0">
        <v-stepper
          v-model="currentStep"
          alt-labels
          class="elevation-0"
        >
          <v-stepper-header>
            <v-stepper-item
              :complete="currentStep > 1"
              :value="1"
              title="Domain Info"
              subtitle="Basic configuration"
            >
              <template v-slot:icon="{ complete }">
                <v-icon v-if="complete" color="success">mdi-check</v-icon>
                <v-icon v-else>mdi-web</v-icon>
              </template>
            </v-stepper-item>

            <v-divider></v-divider>

            <v-stepper-item
              :complete="currentStep > 2"
              :value="2"
              title="Configuration"
              subtitle="Access, routing & authentication"
            >
              <template v-slot:icon="{ complete }">
                <v-icon v-if="complete" color="success">mdi-check</v-icon>
                <v-icon v-else>mdi-cog</v-icon>
              </template>
            </v-stepper-item>
          </v-stepper-header>

          <v-stepper-window>
            <!-- Step 1: Domain Info -->
            <v-stepper-window-item :value="1">
              <v-card-text class="pt-4">
                <div class="text-center mb-4">
                  <v-icon size="48" color="purple" class="mb-3">mdi-web</v-icon>
                  <h3 class="text-h6 mb-2">Domain Information</h3>
                  <p class="text-body-2 text-medium-emphasis">
                    Enter the domain name that should be managed by this DNS server
                  </p>
                </div>

                <v-form ref="step1Form" v-model="step1Valid">
                  <v-text-field
                    v-model="domainData.name"
                    label="Domain Name"
                    variant="outlined"
                    prepend-inner-icon="mdi-web"
                    placeholder="subdomain.example.com"
                    hint="Must have at least 3 parts (e.g., app.example.com, home.mydomain.org) - you need DNS control of the parent domain"
                    persistent-hint
                    :rules="domainRules"
                    @input="onDomainNameChange"
                    required
                  ></v-text-field>

                  <!-- Real-time DNS Validation -->
                  <v-card variant="outlined" class="mt-4">
                    <v-card-title class="text-subtitle-1">
                      <v-icon class="me-2" color="info">mdi-dns</v-icon>
                      DNS Validation
                    </v-card-title>
                    <v-card-text>
                      <v-list density="compact">
                        <!-- NS Records -->
                        <v-list-item>
                          <template v-slot:prepend>
                            <v-icon 
                              :color="getDnsCheckColor('nsRecords')" 
                              :icon="getDnsCheckIcon('nsRecords')"
                              size="small"
                            ></v-icon>
                          </template>
                          <v-list-item-title class="text-body-2">NS Records & Resolution</v-list-item-title>
                          <v-list-item-subtitle class="text-caption">{{ getDnsCheckMessage('nsRecords') }}</v-list-item-subtitle>
                        </v-list-item>
                      </v-list>

                      <div v-if="dnsValidationInProgress" class="text-center mt-2">
                        <v-chip size="small" color="info" variant="tonal">
                          <v-icon start size="small">mdi-loading mdi-spin</v-icon>
                          Checking DNS...
                        </v-chip>
                      </div>
                    </v-card-text>
                  </v-card>

                  <!-- DNS Setup Instructions - Only show after failed validation -->
                  <v-card v-if="shouldShowDnsSetupInstructions()" variant="outlined" color="info" class="mt-4">
                    <v-card-title class="text-subtitle-1">
                      <v-icon class="me-2" color="info">mdi-information</v-icon>
                      DNS Setup Required
                    </v-card-title>
                    <v-card-text>
                      <p class="text-body-2 mb-3">
                        To delegate DNS control to this gateway, create the following NS record in your parent domain:
                      </p>
                      
                      <v-card variant="outlined" color="primary" class="pa-3 mb-3" style="background-color: #f5f5f5; border: 1px solid #ddd;">
                        <div class="d-flex align-center justify-space-between">
                          <div class="font-mono text-body-1" style="color: #333;">
                            <strong>{{ getDnsSetupInstructions().name }}</strong> IN NS <strong>{{ getDnsSetupInstructions().target }}</strong>
                          </div>
                          <v-btn 
                            icon="mdi-content-copy" 
                            variant="outlined" 
                            size="small"
                            color="primary"
                            @click="copyDnsRecord"
                            title="Copy DNS record"
                          ></v-btn>
                        </div>
                      </v-card>

                      <v-alert type="info" variant="tonal" density="compact">
                        <v-icon start>mdi-lightbulb</v-icon>
                        <strong>How to:</strong> Add this NS record in your DNS management interface for <strong>{{ getParentDomain() }}</strong>
                      </v-alert>
                    </v-card-text>
                  </v-card>

                  <v-textarea
                    v-model="domainData.description"
                    label="Description (Optional)"
                    variant="outlined"
                    prepend-inner-icon="mdi-text"
                    placeholder="Describe the purpose of this domain"
                    hint="Optional description for this domain"
                    persistent-hint
                    rows="3"
                    auto-grow
                    class="mt-4"
                  ></v-textarea>
                </v-form>
              </v-card-text>
            </v-stepper-window-item>

            <!-- Step 2: Access & Routing Options -->
            <v-stepper-window-item :value="2">
              <v-card-text class="pt-4">
                <div class="text-center mb-4">
                  <v-icon size="48" color="purple" class="mb-3">mdi-cog</v-icon>
                  <h3 class="text-h6 mb-2">Access & Routing Configuration</h3>
                  <p class="text-body-2 text-medium-emphasis">
                    Configure access control and routing options for this domain
                  </p>
                </div>

                <v-form ref="step2Form">
                  <v-card variant="outlined" class="mb-4">
                    <v-card-title class="text-subtitle-1">
                      <v-icon class="me-2" color="warning">mdi-network-outline</v-icon>
                      Network Access Control
                    </v-card-title>
                    <v-card-text>
                      <v-switch
                        v-model="domainData.localNetworkOnly"
                        label="Restrict access to local networks only"
                        color="warning"
                        hint="Only allow connections from private network ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x)"
                        persistent-hint
                      ></v-switch>
                      
                      <div v-if="domainData.localNetworkOnly" class="mt-3">
                        <v-alert
                          type="info"
                          variant="tonal"
                          density="compact"
                          class="text-caption"
                        >
                          <v-icon start>mdi-information</v-icon>
                          This domain will only be accessible from local network addresses. External traffic will be blocked.
                        </v-alert>
                      </div>
                    </v-card-text>
                  </v-card>

                  <v-card variant="outlined">
                    <v-card-title class="text-subtitle-1">
                      <v-icon class="me-2" color="primary">mdi-swap-horizontal</v-icon>
                      Gateway Redirection
                    </v-card-title>
                    <v-card-text>
                      <v-switch
                        v-model="domainData.redirectToGateway"
                        label="Redirect all requests to another gateway"
                        color="primary"
                        hint="Forward all traffic for this domain to a different gateway server"
                        persistent-hint
                      ></v-switch>
                      
                      <div v-if="domainData.redirectToGateway" class="mt-4">
                        <v-text-field
                          v-model="domainData.redirect.target"
                          label="Gateway Target"
                          variant="outlined"
                          prepend-inner-icon="mdi-server-network"
                          placeholder="other-gateway.example.com"
                          hint="The target gateway server to redirect traffic to"
                          persistent-hint
                          :rules="redirectTargetRules"
                          class="mb-4"
                        ></v-text-field>

                        <v-row>
                          <v-col cols="12" md="4">
                            <v-text-field
                              v-model.number="domainData.redirect.http_port"
                              label="HTTP Port"
                              variant="outlined"
                              type="number"
                              prepend-inner-icon="mdi-web"
                              hint="HTTP port on target gateway"
                              persistent-hint
                              :rules="portRules"
                            ></v-text-field>
                          </v-col>
                          <v-col cols="12" md="4">
                            <v-text-field
                              v-model.number="domainData.redirect.https_port"
                              label="HTTPS Port"
                              variant="outlined"
                              type="number"
                              prepend-inner-icon="mdi-lock"
                              hint="HTTPS port on target gateway"
                              persistent-hint
                              :rules="portRules"
                            ></v-text-field>
                          </v-col>
                          <v-col cols="12" md="4">
                            <v-text-field
                              v-model.number="domainData.redirect.dns_port"
                              label="DNS Port"
                              variant="outlined"
                              type="number"
                              prepend-inner-icon="mdi-dns"
                              hint="DNS port on target gateway"
                              persistent-hint
                              :rules="portRules"
                            ></v-text-field>
                          </v-col>
                        </v-row>

                        <v-alert
                          type="warning"
                          variant="tonal"
                          density="compact"
                          class="text-caption mt-3"
                        >
                          <v-icon start>mdi-alert</v-icon>
                          When enabled, all requests to this domain will be forwarded to the specified gateway with the configured ports.
                        </v-alert>
                      </div>
                    </v-card-text>
                  </v-card>
                </v-form>

                <!-- Authentication Configuration (only when needed) -->
                <div v-if="showAuthConfiguration" class="mt-4">
                  <v-alert
                    type="info"
                    variant="tonal"
                    class="mb-4"
                  >
                    <strong>Authentication Required:</strong> This domain needs an authentication server route to handle user login. Configure the hostname for the built-in OAuth server below.
                  </v-alert>

                  <v-text-field
                    v-model="domainData.authHostname"
                    label="Authentication Route Hostname"
                    variant="outlined"
                    prepend-inner-icon="mdi-shield-account"
                    placeholder="auth, login, oauth"
                    hint="Subdomain for the authentication server (e.g., 'auth' creates auth.example.com)"
                    persistent-hint
                    :rules="authHostnameRules"
                    @input="validateStep2"
                    required
                  >
                    <template v-slot:append-inner>
                      <span class="text-caption text-medium-emphasis">.{{ domainData.name }}</span>
                    </template>
                  </v-text-field>

                  <v-card variant="outlined" class="mt-3">
                    <v-card-title class="text-subtitle-1">
                      <v-icon class="me-2" color="purple">mdi-shield-account</v-icon>
                      Authentication Route Preview
                    </v-card-title>
                    <v-card-text>
                      <v-list density="compact">
                        <v-list-item>
                          <v-list-item-title>Full URL</v-list-item-title>
                          <v-list-item-subtitle>
                            <code>{{ domainData.authHostname || 'auth' }}.{{ domainData.name || 'example.com' }}</code>
                          </v-list-item-subtitle>
                        </v-list-item>
                        <v-list-item>
                          <v-list-item-title>Target</v-list-item-title>
                          <v-list-item-subtitle>
                            <code>@auth</code> (Built-in OAuth server)
                          </v-list-item-subtitle>
                        </v-list-item>
                      </v-list>
                    </v-card-text>
                  </v-card>
                </div>
              </v-card-text>
            </v-stepper-window-item>


          </v-stepper-window>
        </v-stepper>
      </v-card-text>

      <v-divider></v-divider>

      <v-card-actions class="justify-space-between pa-4">
        <v-btn
          variant="text"
          @click="closeWizard"
          :disabled="saving"
        >
          Cancel
        </v-btn>

        <div class="d-flex gap-2">
          <v-btn
            v-if="currentStep > 1"
            variant="outlined"
            @click="previousStep"
            :disabled="saving"
          >
            Previous
          </v-btn>

          <v-btn
            v-if="currentStep === 1"
            color="purple"
            variant="outlined"
            @click="nextStep"
            :disabled="!step1Valid"
          >
            Next
          </v-btn>

          <v-btn
            v-if="currentStep === 2"
            color="purple"
            @click="saveDomain"
            :loading="saving"
            :disabled="!step2Valid"
          >
            Create Domain
          </v-btn>
        </div>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import { apiRequest, apiGet } from '../../utils/api.js'

export default {
  name: 'DomainWizard',
  props: {
    modelValue: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:modelValue', 'domain-saved'],
  data() {
    return {
      currentStep: 1,
      step1Valid: false,
      saving: false,
      domainData: {
        name: '',
        description: '',
        localNetworkOnly: false,
        redirectToGateway: false,
        redirect: {
          target: '',
          http_port: 10080,
          https_port: 10443,
          dns_port: 10053
        },
        authHostname: 'auth'
      },
      dnsValidation: {
        nsRecords: { status: 'pending', message: 'Enter a domain name to check DNS', records: [] }
      },
      gatewayNsTarget: null, // Will be populated from gateway config
      dnsValidationInProgress: false,
      dnsValidationTimeout: null,
      domainRules: [
        v => !!v || 'Domain name is required',
        v => /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(v) || 'Invalid domain format',
        v => v.length <= 253 || 'Domain name too long',
        v => {
          const parts = v ? v.split('.') : []
          return parts.length >= 3 || 'Domain must have at least 3 parts (e.g., subdomain.example.com) - you need DNS control of the parent domain'
        }
      ],
      redirectTargetRules: [
        v => !this.domainData.redirectToGateway || !!v || 'Gateway target is required when redirection is enabled',
        v => !v || /^[a-zA-Z0-9.-]+$/.test(v) || 'Invalid hostname format (no protocol, just hostname)'
      ],
      portRules: [
        v => v >= 1 && v <= 65535 || 'Port must be between 1 and 65535'
      ],
      authHostnameRules: [
        v => !!v || 'Authentication hostname is required',
        v => /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/i.test(v) || 'Invalid hostname format (letters, numbers, hyphens only)'
      ]
    }
  },
  mounted() {
    this.fetchGatewayNsTarget()
  },
  computed: {
    localShow: {
      get() {
        return this.modelValue
      },
      set(value) {
        this.$emit('update:modelValue', value)
      }
    },
    step2Valid() {
      // Step 2 is valid if: all basic validation passes AND (auth not needed OR auth is valid)
      const basicValid = this.step1Valid && 
        (!this.domainData.redirectToGateway || (this.domainData.redirect.target && 
          this.redirectTargetRules.every(rule => rule(this.domainData.redirect.target) === true)))
      
      if (!this.showAuthConfiguration) {
        return basicValid
      }
      
      // If auth is needed, also validate auth hostname
      const authValid = this.domainData.authHostname && 
        this.authHostnameRules.every(rule => rule(this.domainData.authHostname) === true)
      
      return basicValid && authValid
    },
    showAuthConfiguration() {
      // Show auth configuration if:
      // 1. NOT using gateway redirect (redirect doesn't need auth route)
      // 2. AND no other domain already has @auth target route (check would need API call)
      // For now, simplified: show if not redirecting to gateway
      return !this.domainData.redirectToGateway
    }
  },
  watch: {
    modelValue(newVal) {
      if (newVal) {
        this.resetWizard()
      }
    }
  },
  methods: {
    resetWizard() {
      this.currentStep = 1
      this.step1Valid = false
      this.saving = false
      this.dnsValidationInProgress = false
      if (this.dnsValidationTimeout) {
        clearTimeout(this.dnsValidationTimeout)
        this.dnsValidationTimeout = null
      }
      this.domainData = {
        name: '',
        description: '',
        localNetworkOnly: false,
        redirectToGateway: false,
        redirect: {
          target: '',
          http_port: 10080,
          https_port: 10443,
          dns_port: 10053
        },
        authHostname: 'auth'
      }
      this.dnsValidation = {
        domain: { status: 'pending', message: 'Enter a domain name to check DNS', records: [] },
        authRoute: { status: 'pending', message: 'Not checked yet', records: [] },
        nsRecords: { status: 'pending', message: 'Enter a domain name to check DNS', records: [] }
      }
    },

    validateStep1() {
      try {
        const name = this.domainData.name || ''
        this.step1Valid = name.length > 0 && this.domainRules.every(rule => rule(name) === true)
        console.log('validateStep1 - name:', name, 'step1Valid:', this.step1Valid)
      } catch (error) {
        console.error('Validation error:', error)
        this.step1Valid = false
      }
    },

    validateStep2() {
      // Step 2 validation is now handled by the computed property
      // This method exists for template compatibility
      console.log('validateStep2 - step2Valid:', this.step2Valid)
    },

    nextStep() {
      if (this.currentStep === 1 && this.step1Valid) {
        this.currentStep = 2
      }
    },

    onDomainNameChange() {
      this.validateStep1()
      
      // Clear existing timeout
      if (this.dnsValidationTimeout) {
        clearTimeout(this.dnsValidationTimeout)
      }

      // Reset DNS validation status
      this.dnsValidation.nsRecords.status = 'pending'
      this.dnsValidation.nsRecords.message = 'Waiting for input...'

      const domainName = this.domainData.name.trim()
      
      // Only validate if domain name is long enough, valid format, and has at least 3 parts
      const domainParts = domainName.split('.')
      if (domainName.length > 3 && /^[a-z0-9.-]+\.[a-z]{2,}$/i.test(domainName) && domainParts.length >= 3) {
        // Debounce DNS validation to avoid too many API calls
        this.dnsValidationTimeout = setTimeout(() => {
          this.validateDomainAndNsRecords()
        }, 1500) // Wait 1.5 seconds after user stops typing
      } else if (domainName.length > 0) {
        this.dnsValidation.nsRecords.message = 'Enter a valid domain name'
      } else {
        this.dnsValidation.nsRecords.message = 'Enter a domain name to check NS records and resolution'
      }
    },

    async validateDomainAndNsRecords() {
      if (!this.domainData.name.trim()) return

      this.dnsValidationInProgress = true
      
      try {
        // Only validate NS records now
        await this.validateNsRecords()
      } finally {
        this.dnsValidationInProgress = false
      }
    },

    previousStep() {
      if (this.currentStep > 1) {
        this.currentStep--
      }
    },

    closeWizard() {
      this.localShow = false
    },

    async saveDomain() {
      console.log('saveDomain called, step2Valid:', this.step2Valid)
      if (!this.step2Valid) {
        console.log('Validation failed, not saving domain')
        return
      }

      this.saving = true
      try {
        console.log('Creating domain via API:', this.domainData)
        
        // Prepare domain data for API
        const domainPayload = {
          name: this.domainData.name.trim().toLowerCase(),
          routes: []
        }

        // Add auth route only if needed (when not using gateway redirect)
        if (this.showAuthConfiguration && this.domainData.authHostname) {
          domainPayload.routes.push({
            hostname: this.domainData.authHostname,
            target: '@auth'
          })
        }

        // Add optional fields if they have values
        if (this.domainData.description) {
          domainPayload.description = this.domainData.description
        }
        if (this.domainData.localNetworkOnly) {
          domainPayload.localNetworkOnly = true
        }
        if (this.domainData.redirectToGateway && this.domainData.redirect.target) {
          domainPayload.redirect = {
            target: this.domainData.redirect.target,
            http_port: this.domainData.redirect.http_port,
            https_port: this.domainData.redirect.https_port,
            dns_port: this.domainData.redirect.dns_port
          }
        }

        // Make API call to create domain with auth route
        console.log('About to make API call with payload:', domainPayload)
        const response = await apiRequest('domains', {
          method: 'POST',
          body: JSON.stringify(domainPayload)
        })
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.status} ${response.statusText}`)
        }
        
        const newDomain = await response.json()
        
        console.log('Domain and auth route created successfully:', newDomain)
        
        // Emit event to parent component
        this.$emit('domain-saved', newDomain)
        
        // Close wizard
        this.closeWizard()
        
      } catch (error) {
        console.error('Error creating domain:', error)
        // You could add error handling here, perhaps emit an error event
        // or show a notification
      } finally {
        this.saving = false
      }
    },

    // DNS Validation Methods



    async validateNsRecords() {
      this.dnsValidation.nsRecords.status = 'checking'
      this.dnsValidation.nsRecords.message = 'Checking NS records and resolution...'

      try {
        // First get the gateway's current IPv4 address for comparison
        const gatewayIpResponse = await apiGet('dns/external/ipv4')
        const expectedIp = gatewayIpResponse.address

        if (!expectedIp || gatewayIpResponse.error) {
          this.dnsValidation.nsRecords.status = 'error'
          this.dnsValidation.nsRecords.message = `Cannot determine gateway IP: ${gatewayIpResponse.error || 'No address available'}`
          this.dnsValidation.nsRecords.records = []
          return
        }

        // Check NS records
        const nsResponse = await apiGet(`dns/lookup?hostname=${encodeURIComponent(this.domainData.name)}&type=NS`)
        
        if (nsResponse.records && nsResponse.records.length > 0) {
          // Filter for NS records and extract names from the value field
          const nsRecords = nsResponse.records.filter(record => record.type === 'NS')
          
          if (nsRecords.length > 0) {
            const nsNames = nsRecords.map(record => record.value)
            
            // Check if this is a gateway-managed domain (has SOA record) or externally delegated
            const soaRecords = nsResponse.records.filter(record => record.type === 'SOA')
            const isGatewayManaged = soaRecords.length > 0
            
            // Check if the NS records point to your gateway (for external delegation)
            const gatewayNsPatterns = [
              /\.myfritz\.net\.?$/,  // FRITZ!Box dynamic DNS
              new RegExp(`^${expectedIp.replace(/\./g, '\\.')}$`)  // Direct IP match
            ]
            
            const isExternalDelegation = nsNames.some(ns => 
              gatewayNsPatterns.some(pattern => pattern.test(ns))
            )
            
            // For gateway-managed domains, check if NS points to a subdomain of the domain itself
            const isGatewayNsSubdomain = nsNames.some(ns => {
              const cleanNs = ns.replace(/\.$/, '') // Remove trailing dot
              const cleanDomain = this.domainData.name.trim()
              return cleanNs.endsWith(`.${cleanDomain}`) || cleanNs === `ns1.${cleanDomain}`
            })

            if (isGatewayManaged) {
              // Domain is already managed by the gateway (has SOA record)
              this.dnsValidation.nsRecords.status = 'success'
              this.dnsValidation.nsRecords.message = `✓ Domain is managed by this gateway (authoritative with SOA record)`
              this.dnsValidation.nsRecords.records = nsNames
              
            } else if (isExternalDelegation) {
              // NS records point to gateway/FRITZ!Box via external delegation
              // Try to check A records but be lenient if they don't resolve yet
              try {
                const aResponse = await apiGet(`dns/lookup?hostname=${encodeURIComponent(this.domainData.name)}&type=A`)
                
                if (aResponse.records && aResponse.records.length > 0) {
                  const aRecords = aResponse.records.filter(record => record.type === 'A')
                  
                  if (aRecords.length > 0) {
                    const ipAddresses = aRecords.map(record => record.value)
                    const matchingRecords = ipAddresses.filter(ip => ip === expectedIp)
                    
                    if (matchingRecords.length > 0) {
                      this.dnsValidation.nsRecords.status = 'success'
                      this.dnsValidation.nsRecords.message = `✓ NS records correctly delegated to gateway → Domain resolves to ${expectedIp}`
                      this.dnsValidation.nsRecords.records = nsNames
                    } else {
                      this.dnsValidation.nsRecords.status = 'warning'
                      this.dnsValidation.nsRecords.message = `✓ NS delegation correct, but A records point to: ${ipAddresses.join(', ')} (expected ${expectedIp})`
                      this.dnsValidation.nsRecords.records = nsNames
                    }
                  } else {
                    // NS delegation looks good, but no A records yet - this is often fine
                    this.dnsValidation.nsRecords.status = 'success'
                    this.dnsValidation.nsRecords.message = `✓ NS records correctly delegated to gateway (${nsNames.join(', ')}) - A records will be served by your gateway`
                    this.dnsValidation.nsRecords.records = nsNames
                  }
                } else {
                  // NS delegation looks good, no A records yet - this is often fine for new domains
                  this.dnsValidation.nsRecords.status = 'success'
                  this.dnsValidation.nsRecords.message = `✓ NS records correctly delegated to gateway (${nsNames.join(', ')}) - DNS propagation may still be in progress`
                  this.dnsValidation.nsRecords.records = nsNames
                }
              } catch (aError) {
                // NS delegation is correct, A record query failed but that's ok
                this.dnsValidation.nsRecords.status = 'success'
                this.dnsValidation.nsRecords.message = `✓ NS records correctly delegated to gateway (${nsNames.join(', ')}) - Gateway will handle DNS resolution`
                this.dnsValidation.nsRecords.records = nsNames
              }
              
            } else if (isGatewayNsSubdomain) {
              // NS records point to a subdomain of the domain itself (like ns1.example.com for example.com)
              // This suggests the domain might be managed by the gateway but SOA wasn't returned
              this.dnsValidation.nsRecords.status = 'success'
              this.dnsValidation.nsRecords.message = `✓ Domain appears to be managed by gateway (NS: ${nsNames.join(', ')})`
              this.dnsValidation.nsRecords.records = nsNames
              
            } else {
              // NS records don't point to gateway - this needs attention
              this.dnsValidation.nsRecords.status = 'warning'
              this.dnsValidation.nsRecords.message = `NS records found but don't point to gateway: ${nsNames.join(', ')} (expected *.myfritz.net, gateway subdomain, or ${expectedIp})`
              this.dnsValidation.nsRecords.records = nsNames
            }
          } else {
            this.dnsValidation.nsRecords.status = 'warning'
            this.dnsValidation.nsRecords.message = 'No NS records found'
            this.dnsValidation.nsRecords.records = []
          }
        } else {
          this.dnsValidation.nsRecords.status = 'warning'
          this.dnsValidation.nsRecords.message = 'No DNS records found'
          this.dnsValidation.nsRecords.records = []
        }
      } catch (error) {
        this.dnsValidation.nsRecords.status = 'error'
        this.dnsValidation.nsRecords.message = `DNS lookup failed: ${error.message}`
        this.dnsValidation.nsRecords.records = []
      }
    },

    getDnsCheckColor(checkType) {
      const status = this.dnsValidation[checkType]?.status
      switch (status) {
        case 'success': return 'success'
        case 'warning': return 'warning'
        case 'error': return 'error'
        case 'checking': return 'info'
        default: return 'grey'
      }
    },

    getDnsCheckIcon(checkType) {
      const status = this.dnsValidation[checkType]?.status
      switch (status) {
        case 'success': return 'mdi-check-circle'
        case 'warning': return 'mdi-alert'
        case 'error': return 'mdi-close-circle'
        case 'checking': return 'mdi-loading mdi-spin'
        default: return 'mdi-help-circle'
      }
    },

    getDnsCheckMessage(checkType) {
      // If no domain name is entered yet, show helpful placeholder
      if (!this.domainData.name || this.domainData.name.length <= 3) {
        return 'Enter a domain name to check NS records and resolution'
      }
      
      return this.dnsValidation[checkType]?.message || 'Not checked yet'
    },

    shouldShowDnsSetupInstructions() {
      // Only show DNS setup instructions if:
      // 1. Domain name is valid (3+ parts)
      // 2. DNS validation has completed (not pending or checking)
      // 3. DNS validation failed or shows a warning (not success)
      
      if (!this.domainData.name || this.domainData.name.split('.').length < 3) {
        return false
      }

      const status = this.dnsValidation.nsRecords.status
      
      // Show instructions only when validation completed with failure/warning
      return status === 'warning' || status === 'error'
    },

    getDnsSetupInstructions() {
      if (!this.domainData.name || this.domainData.name.split('.').length < 3) {
        return null
      }

      const domainParts = this.domainData.name.split('.')
      const subdomain = domainParts[0]
      
      // Use the fetched NS target or fall back to a generic pattern
      let nsTarget = this.gatewayNsTarget || 'your-fritz-hostname.myfritz.net.'
      
      // Ensure it ends with a dot for proper DNS format
      if (!nsTarget.endsWith('.')) {
        nsTarget += '.'
      }
      
      return {
        name: subdomain,
        target: nsTarget,
        fullRecord: `${subdomain} IN NS ${nsTarget}`
      }
    },

    getParentDomain() {
      if (!this.domainData.name) return ''
      const parts = this.domainData.name.split('.')
      return parts.slice(1).join('.')
    },

    async copyDnsRecord() {
      const instructions = this.getDnsSetupInstructions()
      if (instructions) {
        try {
          await navigator.clipboard.writeText(instructions.fullRecord)
          // You could add a toast notification here
          console.log('DNS record copied to clipboard')
        } catch (error) {
          console.error('Failed to copy DNS record:', error)
        }
      }
    },

    async fetchGatewayNsTarget() {
      try {
        // Get the DNS configuration to extract the IPv4 Source Address
        const response = await apiGet('dns/external/ipv4')
        if (response.source) {
          // Use the IPv4 Source Address as the NS target
          this.gatewayNsTarget = response.source
        }
      } catch (error) {
        console.warn('Could not fetch gateway NS target:', error)
      }
    }
  }
}
</script>

<style scoped>
.v-stepper {
  box-shadow: none !important;
}

.v-stepper-header {
  box-shadow: none;
}
</style>