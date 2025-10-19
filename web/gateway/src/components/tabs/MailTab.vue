<template>
  <v-container fluid>
    <v-row>
      <v-col cols="12">
        <v-card>
          <v-card-title class="text-h6 d-flex align-center">
            <v-icon class="me-2" color="orange">mdi-email</v-icon>
            Mail Configuration
          </v-card-title>
          <v-card-subtitle>
            Configure SMTP settings for password reset emails and notifications
          </v-card-subtitle>
          
          <v-card-text>
            <v-alert
              v-if="!mailConfig.enabled"
              type="info"
              variant="tonal"
              class="mb-4"
            >
              Configure SMTP settings to enable password reset emails and notifications.
            </v-alert>

            <v-form ref="mailForm" @submit.prevent="saveMailConfig">
              <v-row>
                <v-col cols="12" md="6">
                  <v-switch
                    v-model="mailConfig.enabled"
                    label="Enable Mail Service"
                    color="primary"
                    hide-details
                  />
                </v-col>
              </v-row>

              <div v-if="mailConfig.enabled">
                <v-divider class="my-6" />
                
                <v-row>
                  <v-col cols="12">
                    <h3 class="text-h6 mb-4">Authentication</h3>
                  </v-col>
                </v-row>

                <v-row>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="mailConfig.email"
                      label="Email Address / Username"
                      type="email"
                      prepend-inner-icon="mdi-account"
                      hint="SMTP authentication username or email address"
                      persistent-hint
                      autocomplete="off"
                      required
                    />
                  </v-col>
                  
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="displayPassword"
                      label="Password"
                      :type="isPasswordRedacted ? 'text' : (showPassword ? 'text' : 'password')"
                      :append-inner-icon="showPasswordToggle ? (showPassword ? 'mdi-eye' : 'mdi-eye-off') : ''"
                      @click:append-inner="showPasswordToggle ? (showPassword = !showPassword) : null"
                      @click="handlePasswordClick"
                      @focus="handlePasswordFocus"
                      @blur="handlePasswordBlur"
                      hint="SMTP authentication password or app-specific password"
                      persistent-hint
                      :autocomplete="isPasswordRedacted || passwordEditingMode ? 'nope' : 'new-password'"
                      :readonly="isPasswordRedacted"
                      :data-lpignore="isPasswordRedacted || passwordEditingMode"
                      :data-form-type="isPasswordRedacted || passwordEditingMode ? 'other' : 'password'"
                      :data-1p-ignore="isPasswordRedacted || passwordEditingMode"
                      :data-bwignore="isPasswordRedacted || passwordEditingMode"
                      :data-kwignore="isPasswordRedacted || passwordEditingMode"
                      :data-chrome-ignore="isPasswordRedacted || passwordEditingMode"
                      :name="isPasswordRedacted || passwordEditingMode ? 'config-field' : 'password'"
                      :id="isPasswordRedacted || passwordEditingMode ? 'config-field' : 'password-field'"
                      :spellcheck="false"
                      required
                    />
                  </v-col>
                </v-row>

                <v-divider class="my-6" />
                
                <v-row>
                  <v-col cols="12">
                    <h3 class="text-h6 mb-4">SMTP Server Configuration</h3>
                  </v-col>
                </v-row>

                <v-row>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="mailConfig.smtp_host"
                      label="SMTP Server"
                      placeholder="smtp.gmail.com"
                      hint="Your SMTP server hostname"
                      persistent-hint
                      autocomplete="off"
                      required
                    />
                  </v-col>
                  
                  <v-col cols="12">
                    <v-radio-group
                      v-model="portSelection"
                      @update:model-value="onPortSelectionChange"
                      label="Port & Security Configuration"
                      class="mt-0"
                    >
                      <v-row>
                        <v-col cols="12" sm="6" md="3">
                          <div class="port-option">
                            <v-radio
                              label="465 (SSL)"
                              value="465-ssl"
                              color="primary"
                            />
                            <div class="text-caption text-medium-emphasis ml-8">Direct SSL connection</div>
                          </div>
                        </v-col>
                        <v-col cols="12" sm="6" md="3">
                          <div class="port-option">
                            <v-radio
                              label="587 (STARTTLS)"
                              value="587-starttls" 
                              color="primary"
                            />
                            <div class="text-caption text-medium-emphasis ml-8">Standard secure SMTP</div>
                          </div>
                        </v-col>
                        <v-col cols="12" sm="6" md="3">
                          <div class="port-option">
                            <v-radio
                              label="25 (Legacy)"
                              value="25-plain"
                              color="primary"
                            />
                            <div class="text-caption text-medium-emphasis ml-8">Unencrypted (not recommended)</div>
                          </div>
                        </v-col>
                        <v-col cols="12" sm="6" md="3">
                          <div class="port-option">
                            <v-radio
                              label="Custom"
                              value="custom"
                              color="primary"
                            />
                            <div class="text-caption text-medium-emphasis ml-8">Manual configuration</div>
                          </div>
                        </v-col>
                      </v-row>
                    </v-radio-group>
                  </v-col>
                </v-row>

                <!-- Custom Port Configuration -->
                <v-row v-if="portSelection === 'custom'">
                  <v-col cols="12" md="4">
                    <v-text-field
                      v-model.number="mailConfig.smtp_port"
                      label="Custom Port"
                      type="number"
                      placeholder="587"
                      required
                    />
                  </v-col>
                  
                  <v-col cols="12" md="4">
                    <v-switch
                      v-model="mailConfig.use_tls"
                      label="Use TLS/SSL"
                      color="primary"
                      hide-details
                    />
                  </v-col>
                </v-row>

                <v-divider class="my-6" />

                <v-row>
                  <v-col cols="12">
                    <h3 class="text-h6 mb-4">From Address Settings</h3>
                  </v-col>
                </v-row>

                <v-row>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="mailConfig.from_email"
                      label="From Email Address"
                      type="email"
                      hint="Email address used as sender for outgoing emails"
                      persistent-hint
                      autocomplete="off"
                      required
                    />
                  </v-col>
                  
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="mailConfig.from_name"
                      label="From Name"
                      placeholder="Gateway System"
                      hint="Display name for outgoing emails"
                      persistent-hint
                      autocomplete="off"
                    />
                  </v-col>
                </v-row>

                <v-row>
                  <v-col cols="12">
                    <v-btn
                      type="submit"
                      color="primary"
                      :loading="saving"
                      class="mr-3"
                    >
                      <v-icon start>mdi-content-save</v-icon>
                      Save Configuration
                    </v-btn>
                    
                    <v-btn
                      @click="testMailConfig"
                      color="secondary"
                      variant="outlined"
                      :loading="testing"
                      :disabled="!isConfigValid"
                    >
                      <v-icon start>mdi-email-send</v-icon>
                      Send Test Email
                    </v-btn>
                  </v-col>
                </v-row>
              </div>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Test Email Dialog -->
    <v-dialog v-model="testDialog" max-width="500">
      <v-card>
        <v-card-title>Send Test Email</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="testEmail"
            label="Test Email Address"
            type="email"
            hint="Enter email address to receive test message"
            persistent-hint
            autocomplete="off"
            autofocus
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="testDialog = false">Cancel</v-btn>
          <v-btn @click="sendTestEmail" color="primary" :loading="testing">
            Send Test
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Snackbar for notifications -->
    <v-snackbar
      v-model="snackbar.show"
      :color="snackbar.color"
      :timeout="4000"
    >
      {{ snackbar.text }}
      <template #actions>
        <v-btn variant="text" @click="snackbar.show = false">
          Close
        </v-btn>
      </template>
    </v-snackbar>
  </v-container>
</template>

<style scoped>
.port-option {
  margin-bottom: 12px;
}

.port-option .text-caption {
  margin-top: -8px;
}
</style>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { apiGet, apiPut, apiPost } from '@/utils/api.js'

// Form ref
const mailForm = ref(null)

// UI state
const showPassword = ref(false)
const saving = ref(false)
const testing = ref(false)
const testDialog = ref(false)
const testEmail = ref('')
const portSelection = ref('587-starttls')
const isPasswordRedacted = ref(false)
const wasPasswordRedacted = ref(false)
const passwordEditingMode = ref(false)

// Snackbar
const snackbar = ref({
  show: false,
  text: '',
  color: 'success'
})

// Mail configuration
const mailConfig = ref({
  enabled: false,
  email: '',
  password: '',
  smtp_host: '',
  smtp_port: 587,
  use_tls: true,
  from_email: '',
  from_name: 'Gateway System'
})

// Port selection handler
const onPortSelectionChange = (selection) => {
  switch (selection) {
    case '465-ssl':
      mailConfig.value.smtp_port = 465
      mailConfig.value.use_tls = true
      break
    case '587-starttls':
      mailConfig.value.smtp_port = 587
      mailConfig.value.use_tls = true
      break
    case '25-plain':
      mailConfig.value.smtp_port = 25
      mailConfig.value.use_tls = false
      break
    case 'custom':
      // Don't change values, let user configure manually
      break
  }
}

// Computed properties
const displayPassword = computed({
  get() {
    if (isPasswordRedacted.value && mailConfig.value.password === '-') {
      return '••••••••••••••••' // 16 dots for redacted password
    }
    return mailConfig.value.password
  },
  set(value) {
    mailConfig.value.password = value
    // If user starts typing, it's no longer redacted
    if (value !== '••••••••••••••••' && value !== '-') {
      isPasswordRedacted.value = false
    }
  }
})

const showPasswordToggle = computed(() => {
  // Hide the toggle button when password is redacted
  return !isPasswordRedacted.value
})

const isConfigValid = computed(() => {
  if (!mailConfig.value.enabled) return false
  if (!mailConfig.value.email) return false
  
  // Consider redacted password ("-") as valid, or any actual password
  if (!mailConfig.value.password || (mailConfig.value.password !== '-' && mailConfig.value.password.trim() === '')) {
    return false
  }
  
  if (!mailConfig.value.smtp_host || !mailConfig.value.smtp_port) return false
  if (!mailConfig.value.from_email) return false
  
  return true
})


// API functions
const loadMailConfig = async () => {
  try {
    const response = await apiGet('mail/config')
    if (response) {
      Object.assign(mailConfig.value, response)
      
      // Check if password is redacted (comes back as "-")
      if (response.password === '-') {
        isPasswordRedacted.value = true
        wasPasswordRedacted.value = true
        passwordEditingMode.value = false
      } else {
        isPasswordRedacted.value = false
        wasPasswordRedacted.value = false
        passwordEditingMode.value = false
      }
      
      // Set port selection based on loaded configuration
      updatePortSelection()
    }
  } catch (error) {
    console.error('Failed to load mail config:', error)
  }
}

const updatePortSelection = () => {
  const port = mailConfig.value.smtp_port
  const useTls = mailConfig.value.use_tls
  
  if (port === 465 && useTls) {
    portSelection.value = '465-ssl'
  } else if (port === 587 && useTls) {
    portSelection.value = '587-starttls'
  } else if (port === 25 && !useTls) {
    portSelection.value = '25-plain'
  } else {
    portSelection.value = 'custom'
  }
}

const saveMailConfig = async () => {
  if (!mailForm.value || !(await mailForm.value.validate()).valid) {
    return
  }

  saving.value = true
  try {
    const response = await apiPut('mail/config', mailConfig.value)
    
    // Update the config with the response to handle any backend changes
    if (response) {
      Object.assign(mailConfig.value, response)
      
      // Check if password is still redacted after save
      if (response.password === '-') {
        isPasswordRedacted.value = true
        wasPasswordRedacted.value = true
        passwordEditingMode.value = false
      } else {
        isPasswordRedacted.value = false
        wasPasswordRedacted.value = false
        passwordEditingMode.value = false
      }
    }
    
    showSnackbar('Mail configuration saved successfully', 'success')
  } catch (error) {
    console.error('Failed to save mail config:', error)
    showSnackbar('Failed to save mail configuration', 'error')
  } finally {
    saving.value = false
  }
}

const testMailConfig = () => {
  testEmail.value = mailConfig.value.email || ''
  testDialog.value = true
}

const sendTestEmail = async () => {
  if (!testEmail.value) return

  testing.value = true
  try {
    await apiPost('mail/test', {
      email: testEmail.value
    })
    showSnackbar('Test email sent successfully', 'success')
    testDialog.value = false
  } catch (error) {
    console.error('Failed to send test email:', error)
    showSnackbar('Failed to send test email', 'error')
  } finally {
    testing.value = false
  }
}

const showSnackbar = (text, color = 'success') => {
  snackbar.value = { show: true, text, color }
}

// Password field handlers
const handlePasswordClick = () => {
  if (isPasswordRedacted.value) {
    // Enable editing mode
    isPasswordRedacted.value = false
    passwordEditingMode.value = true
    mailConfig.value.password = ''
  }
}

const handlePasswordFocus = () => {
  if (isPasswordRedacted.value) {
    // Enable editing mode
    isPasswordRedacted.value = false
    passwordEditingMode.value = true
    mailConfig.value.password = ''
  }
}

const handlePasswordBlur = () => {
  // If user leaves field empty and it was originally redacted, restore redacted state
  if (!isPasswordRedacted.value && wasPasswordRedacted.value && (!mailConfig.value.password || mailConfig.value.password.trim() === '')) {
    isPasswordRedacted.value = true
    passwordEditingMode.value = false
    mailConfig.value.password = '-'
  } else if (!isPasswordRedacted.value && mailConfig.value.password && mailConfig.value.password.trim() !== '') {
    // User entered a real password, exit editing mode but stay in normal password mode
    passwordEditingMode.value = false
  }
}

// Lifecycle
onMounted(async () => {
  await loadMailConfig()
})
</script>