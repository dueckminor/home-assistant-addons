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
              title="Options"
              subtitle="Access & routing settings"
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
                    placeholder="example.com"
                    hint="Enter a valid domain name (e.g., example.com, subdomain.example.org)"
                    persistent-hint
                    :rules="domainRules"
                    @input="validateStep1"
                    required
                  ></v-text-field>

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
                      
                      <v-text-field
                        v-if="domainData.redirectToGateway"
                        v-model="domainData.gatewayTarget"
                        label="Gateway Target"
                        variant="outlined"
                        prepend-inner-icon="mdi-server-network"
                        class="mt-4"
                        placeholder="https://other-gateway.example.com"
                        hint="The target gateway server to redirect traffic to"
                        persistent-hint
                        :rules="gatewayTargetRules"
                      ></v-text-field>
                      
                      <div v-if="domainData.redirectToGateway" class="mt-3">
                        <v-alert
                          type="warning"
                          variant="tonal"
                          density="compact"
                          class="text-caption"
                        >
                          <v-icon start>mdi-alert</v-icon>
                          When enabled, all requests to this domain will be forwarded to the specified gateway instead of normal routing.
                        </v-alert>
                      </div>
                    </v-card-text>
                  </v-card>
                </v-form>

                <!-- Summary -->
                <v-card variant="outlined" class="mt-4">
                  <v-card-title class="text-subtitle-1">
                    <v-icon class="me-2">mdi-information</v-icon>
                    Domain Summary
                  </v-card-title>
                  <v-card-text>
                    <v-list density="compact">
                      <v-list-item>
                        <v-list-item-title>Domain Name</v-list-item-title>
                        <v-list-item-subtitle>{{ domainData.name || 'Not specified' }}</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item v-if="domainData.description">
                        <v-list-item-title>Description</v-list-item-title>
                        <v-list-item-subtitle>{{ domainData.description }}</v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <v-list-item-title>Network Access</v-list-item-title>
                        <v-list-item-subtitle>
                          <v-chip :color="domainData.localNetworkOnly ? 'warning' : 'success'" size="small">
                            {{ domainData.localNetworkOnly ? 'Local Only' : 'Public Access' }}
                          </v-chip>
                        </v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item>
                        <v-list-item-title>Routing</v-list-item-title>
                        <v-list-item-subtitle>
                          <v-chip :color="domainData.redirectToGateway ? 'primary' : 'default'" size="small">
                            {{ domainData.redirectToGateway ? 'Gateway Redirect' : 'Normal Routing' }}
                          </v-chip>
                        </v-list-item-subtitle>
                      </v-list-item>
                      <v-list-item v-if="domainData.redirectToGateway && domainData.gatewayTarget">
                        <v-list-item-title>Gateway Target</v-list-item-title>
                        <v-list-item-subtitle>{{ domainData.gatewayTarget }}</v-list-item-subtitle>
                      </v-list-item>
                    </v-list>
                  </v-card-text>
                </v-card>
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
            v-if="currentStep < 2"
            color="purple"
            variant="outlined"
            @click="nextStep"
            :disabled="!step1Valid"
          >
            Next
          </v-btn>

          <v-btn
            v-else
            color="purple"
            @click="saveDomain"
            :loading="saving"
            :disabled="!step1Valid"
          >
            Create Domain
          </v-btn>
        </div>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import { apiRequest } from '../../utils/api.js'

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
        gatewayTarget: ''
      },
      domainRules: [
        v => !!v || 'Domain name is required',
        v => /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(v) || 'Invalid domain format',
        v => v.length <= 253 || 'Domain name too long'
      ],
      gatewayTargetRules: [
        v => !this.domainData.redirectToGateway || !!v || 'Gateway target is required when redirection is enabled',
        v => !v || /^https?:\/\/[a-zA-Z0-9.-]+(\:[0-9]+)?(\/.*)?$/.test(v) || 'Invalid gateway URI format (must start with http:// or https://)'
      ]
    }
  },
  computed: {
    localShow: {
      get() {
        return this.modelValue
      },
      set(value) {
        this.$emit('update:modelValue', value)
      }
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
      this.domainData = {
        name: '',
        description: '',
        localNetworkOnly: false,
        redirectToGateway: false,
        gatewayTarget: ''
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

    nextStep() {
      if (this.currentStep === 1 && this.step1Valid) {
        this.currentStep = 2
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
      console.log('saveDomain called, step1Valid:', this.step1Valid)
      if (!this.step1Valid) {
        console.log('Validation failed, not saving domain')
        return
      }

      this.saving = true
      try {
        console.log('Creating domain via API:', this.domainData)
        
        // Prepare domain data for API
        const domainPayload = {
          name: this.domainData.name.trim().toLowerCase()
        }

        // Add optional fields if they have values
        if (this.domainData.description) {
          domainPayload.description = this.domainData.description
        }
        if (this.domainData.localNetworkOnly) {
          domainPayload.localNetworkOnly = true
        }
        if (this.domainData.redirectToGateway) {
          domainPayload.redirectToGateway = true
          if (this.domainData.gatewayTarget) {
            domainPayload.gatewayTarget = this.domainData.gatewayTarget
          }
        }

        // Make API call to create domain
        console.log('About to make API call with payload:', domainPayload)
        const response = await apiRequest('domains', {
          method: 'POST',
          body: JSON.stringify(domainPayload)
        })
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.status} ${response.statusText}`)
        }
        
        const newDomain = await response.json()
        
        console.log('Domain created successfully:', newDomain)
        
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