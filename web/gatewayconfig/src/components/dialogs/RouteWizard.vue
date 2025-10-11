<template>
  <v-dialog v-model="dialog" max-width="900" persistent>
    <v-card>
      <v-card-title class="text-h5 d-flex align-center pa-4">
        <v-icon class="me-2" color="primary">mdi-creation</v-icon>
        {{ editMode ? 'Edit Route' : 'Create New Route' }}
        <v-spacer></v-spacer>
        <v-btn icon="mdi-close" variant="text" @click="closeDialog"></v-btn>
      </v-card-title>

      <v-card-subtitle class="pb-2">
        Step {{ currentStep }} of {{ totalSteps }}: {{ stepTitles[currentStep - 1] }}
      </v-card-subtitle>

      <!-- Progress Indicator -->
      <v-card-text class="pb-0">
        <v-stepper v-model="currentStep" alt-labels>
          <v-stepper-header>
            <v-stepper-item 
              value="1" 
              title="Basic Info"
              :complete="currentStep > 1"
              :color="getStepColor(1)"
            >
              <template v-slot:icon>
                <v-icon>mdi-information</v-icon>
              </template>
            </v-stepper-item>

            <v-divider></v-divider>

            <v-stepper-item 
              value="2" 
              title="Security"
              :complete="currentStep > 2"
              :color="getStepColor(2)"
            >
              <template v-slot:icon>
                <v-icon>mdi-shield-lock</v-icon>
              </template>
            </v-stepper-item>

            <v-divider></v-divider>

            <v-stepper-item 
              value="3" 
              title="Test"
              :complete="currentStep > 3"
              :color="getStepColor(3)"
            >
              <template v-slot:icon>
                <v-icon>mdi-network-outline</v-icon>
              </template>
            </v-stepper-item>

            <v-divider></v-divider>

            <v-stepper-item 
              value="4" 
              title="Summary"
              :complete="false"
              :color="getStepColor(4)"
            >
              <template v-slot:icon>
                <v-icon>mdi-check-circle</v-icon>
              </template>
            </v-stepper-item>
          </v-stepper-header>
        </v-stepper>
      </v-card-text>

      <v-divider></v-divider>

        <!-- Step Content -->
      <v-card-text class="pa-6 wizard-content">
        <!-- Step 1: Basic Information -->
        <div v-if="currentStep == 1" key="step1">
          <h3 class="text-h6 mb-4">Basic Route Information</h3>
          <v-row>
            <v-col cols="12">
              <v-text-field
                v-model="routeData.hostname"
                label="Hostname"
                variant="outlined"
                prepend-inner-icon="mdi-web"
                placeholder="api.example.com"
                :rules="hostnameRules"
                hint="The hostname that should be routed (subdomain of your domain)"
                persistent-hint
                @update:modelValue="validateStep1"
              ></v-text-field>
            </v-col>
            <v-col cols="12">
              <v-text-field
                v-model="routeData.target"
                label="Target URI"
                variant="outlined"
                prepend-inner-icon="mdi-link"
                placeholder="https://fritz.box or http://something.local"
                :rules="targetRules"
                hint="The target URI to route traffic to (including protocol)"
                persistent-hint
                @update:modelValue="validateStep1"
              ></v-text-field>
            </v-col>
          </v-row>
        </div>

        <!-- Step 2: Security Options -->
        <div v-if="currentStep == 2">
          <h3 class="text-h6 mb-4">Security & TLS Options</h3>
          
          <v-row>
            <v-col cols="12">
              <v-card variant="outlined">
                <v-card-title class="text-subtitle-1">
                  <v-icon class="me-2">mdi-certificate</v-icon>
                  TLS Configuration
                </v-card-title>
                <v-card-text>
                  <v-switch
                    v-model="routeData.ignoreInvalidTls"
                    color="warning"
                    label="Ignore invalid TLS certificates"
                    hint="Allow connections to targets with self-signed or invalid certificates"
                    persistent-hint
                  ></v-switch>
                  
                  <v-switch
                    v-model="routeData.forceHttps"
                    color="success"
                    label="Force HTTPS"
                    hint="Automatically redirect HTTP requests to HTTPS"
                    persistent-hint
                    class="mt-2"
                  ></v-switch>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>

          <v-row class="mt-4">
            <v-col cols="12">
              <v-card variant="outlined">
                <v-card-title class="text-subtitle-1">
                  <v-icon class="me-2">mdi-shield-account</v-icon>
                  Authorization
                </v-card-title>
                <v-card-text>
                  <v-switch
                    v-model="routeData.requireAuth"
                    color="primary"
                    label="Require additional authorization"
                    hint="Force users to authenticate before accessing this route"
                    persistent-hint
                  ></v-switch>
                  
                  <v-text-field
                    v-if="routeData.requireAuth"
                    v-model="routeData.authRealm"
                    label="Authentication Realm"
                    variant="outlined"
                    class="mt-4"
                    placeholder="Protected API"
                    hint="Display name for the authentication prompt"
                    persistent-hint
                  ></v-text-field>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>
        </div>

        <!-- Step 3: Connectivity Test -->
        <div v-if="currentStep == 3">
          <h3 class="text-h6 mb-4">Test Target Connectivity</h3>
          
          <v-card variant="tonal" color="info" class="mb-4">
            <v-card-text>
              <div class="d-flex align-center">
                <v-icon class="me-3">mdi-information</v-icon>
                <div>
                  <div class="text-subtitle-2">Testing connection to URI:</div>
                  <div class="text-body-2 font-weight-bold">{{ routeData.target }}</div>
                </div>
              </div>
            </v-card-text>
          </v-card>

          <div class="text-center">
            <v-btn
              :loading="testing"
              :disabled="!routeData.target"
              color="primary"
              size="large"
              @click="testConnection"
              class="mb-4"
            >
              <v-icon start>mdi-play</v-icon>
              Test Connection
            </v-btn>
          </div>

          <!-- Test Results -->
          <v-card v-if="testResult" :color="testResult.success ? 'success' : 'error'" variant="tonal">
            <v-card-text>
              <div class="d-flex align-center mb-2">
                <v-icon class="me-2" size="large">
                  {{ testResult.success ? 'mdi-check-circle' : 'mdi-alert-circle' }}
                </v-icon>
                <span class="text-h6">
                  {{ testResult.success ? 'Connection Successful' : 'Connection Failed' }}
                </span>
              </div>
              
              <div class="text-body-2">
                <strong>Status:</strong> {{ testResult.status }}<br>
                <strong>Response Time:</strong> {{ testResult.responseTime }}ms<br>
                <strong>Details:</strong> {{ testResult.message }}
              </div>
              
              <div v-if="testResult.warnings && testResult.warnings.length > 0" class="mt-3">
                <v-alert type="warning" variant="tonal" class="mb-0">
                  <div class="text-subtitle-2 mb-1">Warnings:</div>
                  <ul class="text-body-2">
                    <li v-for="warning in testResult.warnings" :key="warning">{{ warning }}</li>
                  </ul>
                </v-alert>
              </div>
            </v-card-text>
          </v-card>

          <v-alert v-if="!testResult" type="info" variant="tonal" class="mt-4">
            Click "Test Connection" to verify that the target server is reachable and responding correctly.
          </v-alert>
        </div>

        <!-- Step 4: Summary -->
        <div v-if="currentStep == 4">
          <h3 class="text-h6 mb-4">Review Route Configuration</h3>
          
          <v-card variant="outlined">
            <v-card-title class="text-subtitle-1">
              <v-icon class="me-2">mdi-dns</v-icon>
              Route Details
            </v-card-title>
            <v-card-text>
              <v-row>
                <v-col cols="6">
                  <div class="text-caption text-medium-emphasis">Hostname</div>
                  <div class="text-body-1 font-weight-medium">{{ routeData.hostname }}</div>
                </v-col>
                <v-col cols="6">
                  <div class="text-caption text-medium-emphasis">Target URI</div>
                  <div class="text-body-1 font-weight-medium">{{ routeData.target }}</div>
                </v-col>
              </v-row>
            </v-card-text>
          </v-card>

          <v-card variant="outlined" class="mt-4">
            <v-card-title class="text-subtitle-1">
              <v-icon class="me-2">mdi-cog</v-icon>
              Configuration Options
            </v-card-title>
            <v-card-text>
              <v-row>
                <v-col cols="12" md="6">
                  <v-chip :color="routeData.ignoreInvalidTls ? 'warning' : 'default'" class="mb-2 mr-2">
                    <v-icon start>{{ routeData.ignoreInvalidTls ? 'mdi-certificate-off' : 'mdi-certificate' }}</v-icon>
                    {{ routeData.ignoreInvalidTls ? 'Ignore Invalid TLS' : 'Validate TLS' }}
                  </v-chip>
                  
                  <v-chip :color="routeData.forceHttps ? 'success' : 'default'" class="mb-2 mr-2">
                    <v-icon start>{{ routeData.forceHttps ? 'mdi-lock' : 'mdi-lock-open' }}</v-icon>
                    {{ routeData.forceHttps ? 'Force HTTPS' : 'Allow HTTP' }}
                  </v-chip>
                </v-col>
                <v-col cols="12" md="6">
                  <v-chip :color="routeData.requireAuth ? 'primary' : 'default'" class="mb-2 mr-2">
                    <v-icon start>{{ routeData.requireAuth ? 'mdi-shield-account' : 'mdi-shield-off' }}</v-icon>
                    {{ routeData.requireAuth ? 'Auth Required' : 'No Auth' }}
                  </v-chip>
                  
                  <div v-if="routeData.requireAuth && routeData.authRealm" class="text-caption mt-1">
                    Realm: {{ routeData.authRealm }}
                  </div>
                </v-col>
              </v-row>
            </v-card-text>
          </v-card>

          <v-card v-if="testResult" :color="testResult.success ? 'success' : 'warning'" variant="tonal" class="mt-4">
            <v-card-text>
              <div class="d-flex align-center">
                <v-icon class="me-2">{{ testResult.success ? 'mdi-check-circle' : 'mdi-alert' }}</v-icon>
                <div>
                  <div class="text-subtitle-2">Connection Test Result</div>
                  <div class="text-body-2">{{ testResult.message }}</div>
                </div>
              </div>
            </v-card-text>
          </v-card>
        </div>
      </v-card-text>

      <v-divider></v-divider>

      <!-- Actions -->
      <v-card-actions class="pa-4">
        <v-btn
          v-if="currentStep > 1"
          variant="outlined"
          @click="previousStep"
        >
          <v-icon start>mdi-chevron-left</v-icon>
          Previous
        </v-btn>
        
        <v-spacer></v-spacer>
        
        <v-btn
          variant="outlined"
          @click="closeDialog"
        >
          Cancel
        </v-btn>
        
        <v-btn
          v-if="currentStep < totalSteps"
          color="primary"
          :disabled="!canProceed"
          @click="nextStep"
        >
          Next
          <v-icon end>mdi-chevron-right</v-icon>
        </v-btn>
        
        <v-btn
          v-else
          color="success"
          :loading="saving"
          @click="saveRoute"
        >
          <v-icon start>mdi-content-save</v-icon>
          {{ editMode ? 'Update Route' : 'Create Route' }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
import { apiRequest } from '../../utils/api.js'

export default {
  name: 'RouteWizard',
  props: {
    modelValue: Boolean,
    domainGuid: String,
    editRoute: Object // If editing existing route
  },
  emits: ['update:modelValue', 'route-saved'],
  data() {
    return {
      currentStep: 1,
      totalSteps: 4,
      stepTitles: ['Basic Info', 'Security', 'Test', 'Summary'],
      testing: false,
      saving: false,
      testResult: null,
      
      routeData: {
        hostname: '',
        target: '',
        ignoreInvalidTls: false,
        forceHttps: true,
        requireAuth: false,
        authRealm: ''
      },
      
      hostnameRules: [
        v => !!v || 'Hostname is required',
        v => /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/.test(v) || 'Invalid hostname format'
      ],
      
      targetRules: [
        v => !!v || 'Target URI is required',
        v => /^https?:\/\/[a-zA-Z0-9.-]+(\:[0-9]+)?(\/.*)?$/.test(v) || 'Invalid URI format (must start with http:// or https://)'
      ],
      
      step1Valid: false
    }
  },
  computed: {
    dialog: {
      get() { return this.modelValue },
      set(value) { this.$emit('update:modelValue', value) }
    },
    
    editMode() {
      return !!this.editRoute
    },
    
    canProceed() {
      switch (parseInt(this.currentStep)) {
        case 1: return this.step1Valid
        case 2: return true // Security options are optional
        case 3: return true // Testing is optional but recommended
        case 4: return true
        default: return false
      }
    }
  },
  
  watch: {
    modelValue(newVal) {
      if (newVal && this.editRoute) {
        // Populate form with existing route data
        this.routeData = { ...this.editRoute }
        this.validateStep1()
      } else if (newVal) {
        // Reset form for new route
        this.resetForm()
      }
    },
    
    'routeData.hostname'() {
      this.validateStep1()
    },
    
    'routeData.target'() {
      this.validateStep1()
    }
  },
  
  mounted() {
    this.validateStep1()
  },
  
  methods: {
    getStepColor(step) {
      const currentStepNum = parseInt(this.currentStep)
      if (currentStepNum === step) return 'primary'
      if (currentStepNum > step) return 'success'
      return 'default'
    },
    
    validateStep1() {
      try {
        const hostname = this.routeData.hostname || ''
        const target = this.routeData.target || ''
        
        const hostnameValid = hostname.length > 0 && this.hostnameRules.every(rule => rule(hostname) === true)
        const targetValid = target.length > 0 && this.targetRules.every(rule => rule(target) === true)
        
        this.step1Valid = hostnameValid && targetValid
      } catch (error) {
        console.error('Validation error:', error)
        this.step1Valid = false
      }
    },
    
    nextStep() {
      const currentStepNum = parseInt(this.currentStep)
      if (currentStepNum < this.totalSteps && this.canProceed) {
        this.currentStep = currentStepNum + 1
      }
    },

    previousStep() {
      const currentStepNum = parseInt(this.currentStep)
      if (currentStepNum > 1) {
        this.currentStep = currentStepNum - 1
        this.testResult = null // Clear test results when going back
      }
    },    async testConnection() {
      this.testing = true
      this.testResult = null
      
      try {
        // TODO: Replace with actual API endpoint
        const response = await apiRequest(`domains/${this.domainGuid}/routes/test`, 'POST', {
          target: this.routeData.target,
          ignoreInvalidTls: this.routeData.ignoreInvalidTls
        })
        
        this.testResult = response
      } catch (error) {
        this.testResult = {
          success: false,
          status: 'Error',
          responseTime: 0,
          message: error.message,
          warnings: []
        }
      } finally {
        this.testing = false
      }
    },
    
    async saveRoute() {
      this.saving = true
      
      try {
        const endpoint = this.editMode 
          ? `domains/${this.domainGuid}/routes/${this.editRoute.guid}`
          : `domains/${this.domainGuid}/routes`
        
        const method = this.editMode ? 'PUT' : 'POST'
        
        const response = await apiRequest(endpoint, {
          method: method,
          body: JSON.stringify(this.routeData)
        })
        
        if (!response.ok) {
          throw new Error(`API request failed: ${response.status} ${response.statusText}`)
        }
        
        // Parse JSON response if available
        let savedRoute
        const contentType = response.headers.get('content-type')
        if (contentType && contentType.includes('application/json')) {
          savedRoute = await response.json()
        } else {
          // If no JSON response, create a mock route object with the data we sent
          savedRoute = {
            ...this.routeData,
            guid: Date.now().toString() // Temporary GUID until API returns proper one
          }
        }
        
        this.$emit('route-saved', savedRoute)
        this.closeDialog()
      } catch (error) {
        console.error('Error saving route:', error)
        // TODO: Show error message to user
      } finally {
        this.saving = false
      }
    },
    
    resetForm() {
      this.currentStep = 1
      this.routeData = {
        hostname: '',
        target: '',
        ignoreInvalidTls: false,
        forceHttps: true,
        requireAuth: false,
        authRealm: ''
      }
      this.testResult = null
      this.step1Valid = false
    },
    
    closeDialog() {
      this.dialog = false
      setTimeout(() => {
        this.resetForm()
      }, 300)
    }
  }
}
</script>

<style scoped>
.v-stepper :deep(.v-stepper-header) {
  box-shadow: none;
}

.wizard-content {
  min-height: 300px;
  padding: 24px;
}

.v-card {
  display: flex;
  flex-direction: column;
}

.v-card-text {
  flex: 1;
  overflow-y: auto;
}
</style>