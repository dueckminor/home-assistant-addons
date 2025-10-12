<template>
  <v-app :theme="isDark ? 'dark' : 'light'">
    <v-main class="auth-background">
      <v-container fluid class="fill-height">
        <v-row justify="center" align="center" class="fill-height">
          <v-col cols="12" sm="8" md="6" lg="4" xl="3">
            <!-- Authentication Success Card -->
            <v-card 
              v-if="authenticated" 
              class="auth-card"
              elevation="8"
            >
              <v-card-title class="text-center pa-6">
                <div class="d-flex flex-column align-center">
                  <v-avatar size="64" color="success" class="mb-4">
                    <v-icon size="32">mdi-check-circle</v-icon>
                  </v-avatar>
                  <h2 class="text-h5 mb-2">Welcome back!</h2>
                  <p class="text-body-1 text-medium-emphasis mb-0">
                    Logged in as <strong>{{ username }}</strong>
                  </p>
                </div>
              </v-card-title>

              <v-card-text class="pa-6 pt-0">
                <v-alert
                  type="success"
                  variant="tonal"
                  class="mb-4"
                >
                  Authentication successful. You can now access protected resources.
                </v-alert>
                
                <div class="text-center">
                  <v-btn
                    color="primary"
                    variant="elevated"
                    size="large"
                    @click="logout"
                    :loading="loggingOut"
                    prepend-icon="mdi-logout"
                  >
                    Logout
                  </v-btn>
                </div>
              </v-card-text>
            </v-card>

            <!-- Login Form Card -->
            <v-card 
              v-else 
              class="auth-card"
              elevation="8"
            >
              <v-card-title class="text-center pa-6">
                <div class="d-flex flex-column align-center">
                  <v-avatar size="64" color="primary" class="mb-4">
                    <v-icon size="32">mdi-shield-account</v-icon>
                  </v-avatar>
                  <h2 class="text-h5 mb-2">Sign In</h2>
                  <p class="text-body-1 text-medium-emphasis mb-0">
                    Enter your credentials to continue
                  </p>
                </div>
              </v-card-title>

              <v-card-text class="pa-6 pt-0">
                <!-- Error Alert -->
                <v-alert
                  v-if="errorMessage"
                  type="error"
                  variant="tonal"
                  class="mb-4"
                  closable
                  @click:close="errorMessage = ''"
                >
                  <v-icon start>mdi-alert-circle</v-icon>
                  {{ errorMessage }}
                </v-alert>

                <v-form ref="loginForm" v-model="formValid" @submit.prevent="login">
                  <v-text-field
                    v-model="username"
                    label="Username"
                    variant="outlined"
                    prepend-inner-icon="mdi-account"
                    :rules="usernameRules"
                    :disabled="loading"
                    autofocus
                    required
                    name="username"
                    id="username"
                    autocomplete="username"
                    @keyup.enter="focusPassword"
                    @input="handleUsernameInput"
                    @change="handleUsernameInput"
                  ></v-text-field>

                  <v-text-field
                    ref="passwordField"
                    v-model="password"
                    label="Password"
                    variant="outlined"
                    prepend-inner-icon="mdi-lock"
                    :append-inner-icon="showPassword ? 'mdi-eye' : 'mdi-eye-off'"
                    :type="showPassword ? 'text' : 'password'"
                    :rules="passwordRules"
                    :disabled="loading"
                    required
                    name="password"
                    id="password"
                    autocomplete="current-password"
                    @click:append-inner="showPassword = !showPassword"
                    @keyup.enter="login"
                    @input="handlePasswordInput"
                    @change="handlePasswordInput"
                  ></v-text-field>

                  <v-checkbox
                    v-model="rememberMe"
                    label="Remember me"
                    color="primary"
                    :disabled="loading"
                    class="mb-4"
                  ></v-checkbox>

                  <v-btn
                    type="submit"
                    color="primary"
                    variant="elevated"
                    size="large"
                    block
                    :loading="loading"
                    :disabled="!formValid"
                    prepend-icon="mdi-login"
                  >
                    Sign In
                  </v-btn>
                </v-form>
              </v-card-text>
            </v-card>

            <!-- Theme Toggle (bottom right) -->
            <v-fab
              location="bottom end"
              size="small"
              color="surface-variant"
              :icon="isDark ? 'mdi-weather-sunny' : 'mdi-weather-night'"
              @click="toggleTheme"
              class="theme-toggle"
            ></v-fab>
          </v-col>
        </v-row>
      </v-container>

      <!-- Loading Overlay -->
      <v-overlay
        v-model="loading"
        class="align-center justify-center"
        contained
      >
        <v-progress-circular
          size="64"
          color="primary"
          indeterminate
        ></v-progress-circular>
      </v-overlay>

      <!-- Status Messages -->
      <v-snackbar
        v-model="showMessage"
        :color="messageColor"
        timeout="4000"
        location="top"
      >
        <v-icon start>{{ messageIcon }}</v-icon>
        {{ messageText }}
        <template v-slot:actions>
          <v-btn
            variant="text"
            @click="showMessage = false"
          >
            Close
          </v-btn>
        </template>
      </v-snackbar>
    </v-main>
  </v-app>
</template>

<script>
import { ref, reactive, computed, onMounted } from 'vue'
import { useTheme } from 'vuetify'

export default {
  name: 'AuthApp',
  setup() {
    const theme = useTheme()
    
    // Reactive state
    const authenticated = ref(false)
    const loading = ref(false)
    const loggingOut = ref(false)
    const formValid = ref(false)
    const showPassword = ref(false)
    const rememberMe = ref(false)
    const isDark = ref(true)
    
    const credentials = reactive({
      username: '',
      password: ''
    })
    
    const oauth = reactive({
      clientId: '',
      responseType: '',
      redirectURI: ''
    })
    
    // Message system
    const showMessage = ref(false)
    const messageText = ref('')
    const messageColor = ref('info')
    const messageIcon = ref('mdi-information')
    const errorMessage = ref('')
    
    // Form validation rules
    const usernameRules = [
      v => !!v || 'Username is required',
      v => v.length >= 2 || 'Username must be at least 2 characters'
    ]
    
    const passwordRules = [
      v => !!v || 'Password is required',
      v => v.length >= 4 || 'Password must be at least 4 characters'
    ]
    
    // Computed properties
    const currentUser = computed(() => credentials.username)
    
    // Methods
    const showNotification = (text, color = 'info', icon = 'mdi-information') => {
      messageText.value = text
      messageColor.value = color
      messageIcon.value = icon
      showMessage.value = true
    }
    
    const toggleTheme = () => {
      isDark.value = !isDark.value
      theme.global.name.value = isDark.value ? 'dark' : 'light'
      localStorage.setItem('auth-theme', isDark.value ? 'dark' : 'light')
    }
    
    const focusPassword = () => {
      // Focus password field when Enter is pressed in username field
      const passwordField = document.querySelector('input[type="password"]')
      if (passwordField) {
        passwordField.focus()
      }
    }
    
    const handleUsernameInput = (event) => {
      credentials.username = event.target.value
    }
    
    const handlePasswordInput = (event) => {
      credentials.password = event.target.value
    }
    
    // Check for autofilled values periodically
    const checkAutofillValues = () => {
      const usernameField = document.getElementById('username')?.querySelector('input')
      const passwordField = document.getElementById('password')?.querySelector('input')
      
      if (usernameField && usernameField.value !== credentials.username) {
        credentials.username = usernameField.value
      }
      
      if (passwordField && passwordField.value !== credentials.password) {
        credentials.password = passwordField.value
      }
    }
    
    const checkAuthStatus = async () => {
      try {
        const response = await fetch('/status')
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        
        const data = await response.json()
        
        if (data.username && data.username !== '') {
          credentials.username = data.username
          authenticated.value = true
        } else {
          authenticated.value = false
        }
      } catch (error) {
        console.error('Failed to check auth status:', error)
        errorMessage.value = 'Failed to connect to authentication server'
        authenticated.value = false
      }
    }
    
    const login = async () => {
      if (!formValid.value) return
      
      loading.value = true
      errorMessage.value = ''
      
      try {
        const response = await fetch('/login', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            username: credentials.username,
            password: credentials.password,
            rememberMe: rememberMe.value
          })
        })
        
        if (response.ok) {
          // Login successful
          authenticated.value = true
          credentials.password = '' // Clear password for security
          
          showNotification('Login successful!', 'success', 'mdi-check-circle')
          
          // Handle OAuth redirect if needed
          if (oauth.redirectURI && oauth.redirectURI !== '') {
            const redirectUrl = new URL('/oauth/authorize', window.location.origin)
            redirectUrl.searchParams.set('client_id', oauth.clientId)
            redirectUrl.searchParams.set('redirect_uri', oauth.redirectURI)
            redirectUrl.searchParams.set('response_type', oauth.responseType)
            
            setTimeout(() => {
              window.location.href = redirectUrl.toString()
            }, 1500) // Give user time to see success message
          }
        } else {
          // Login failed
          const errorData = await response.json().catch(() => ({}))
          
          if (response.status === 401) {
            errorMessage.value = 'Invalid username or password'
          } else if (response.status === 429) {
            errorMessage.value = 'Too many login attempts. Please try again later.'
          } else {
            errorMessage.value = errorData.message || `Login failed (${response.status})`
          }
        }
      } catch (error) {
        console.error('Login error:', error)
        errorMessage.value = 'Connection failed. Please check your network and try again.'
      } finally {
        loading.value = false
      }
    }
    
    const logout = async () => {
      loggingOut.value = true
      errorMessage.value = ''
      
      try {
        const response = await fetch('/logout', {
          method: 'POST'
        })
        
        if (response.ok || response.status === 202) {
          authenticated.value = false
          credentials.username = ''
          credentials.password = ''
          showNotification('Logged out successfully', 'info', 'mdi-logout')
        } else {
          throw new Error(`Logout failed: ${response.status}`)
        }
      } catch (error) {
        console.error('Logout error:', error)
        errorMessage.value = 'Logout failed. Please try again.'
      } finally {
        loggingOut.value = false
      }
    }
    
    const parseUrlParameters = () => {
      const url = new URL(window.location.href)
      oauth.redirectURI = url.searchParams.get('redirect_uri') || ''
      oauth.clientId = url.searchParams.get('client_id') || ''
      oauth.responseType = url.searchParams.get('response_type') || ''
      
      console.log('OAuth parameters:', oauth)
    }
    
    const initializeTheme = () => {
      const savedTheme = localStorage.getItem('auth-theme')
      if (savedTheme) {
        isDark.value = savedTheme === 'dark'
      } else {
        // Default to system preference
        isDark.value = window.matchMedia('(prefers-color-scheme: dark)').matches
      }
      theme.global.name.value = isDark.value ? 'dark' : 'light'
    }
    
    // Lifecycle
    onMounted(async () => {
      initializeTheme()
      parseUrlParameters()
      await checkAuthStatus()
      
      // Set up periodic autofill detection
      const autofillInterval = setInterval(checkAutofillValues, 100)
      
      // Clear interval after 5 seconds (autofill usually happens quickly)
      setTimeout(() => {
        clearInterval(autofillInterval)
      }, 5000)
      
      // Also check on focus events
      setTimeout(() => {
        const usernameField = document.getElementById('username')?.querySelector('input')
        const passwordField = document.getElementById('password')?.querySelector('input')
        
        if (usernameField) {
          usernameField.addEventListener('focus', checkAutofillValues)
          usernameField.addEventListener('blur', checkAutofillValues)
        }
        
        if (passwordField) {
          passwordField.addEventListener('focus', checkAutofillValues)
          passwordField.addEventListener('blur', checkAutofillValues)
        }
      }, 500)
    })
    
    return {
      // State
      authenticated,
      loading,
      loggingOut,
      formValid,
      showPassword,
      rememberMe,
      isDark,
      credentials,
      oauth,
      showMessage,
      messageText,
      messageColor,
      messageIcon,
      errorMessage,
      
      // Computed
      username: computed(() => credentials.username),
      password: computed({
        get: () => credentials.password,
        set: (value) => { credentials.password = value }
      }),
      
      // Validation
      usernameRules,
      passwordRules,
      
      // Methods
      login,
      logout,
      focusPassword,
      toggleTheme,
      showNotification,
      handleUsernameInput,
      handlePasswordInput
    }
  }
}
</script>

<style scoped>
.auth-background {
  background: linear-gradient(135deg, rgb(var(--v-theme-primary)) 0%, rgb(var(--v-theme-secondary)) 100%);
  min-height: 100vh;
}

.auth-card {
  backdrop-filter: blur(10px);
  background: rgba(var(--v-theme-surface), 0.95);
  border-radius: 16px !important;
  border: 1px solid rgba(var(--v-theme-outline), 0.2);
  max-width: 400px;
  width: 100%;
}

.theme-toggle {
  position: fixed !important;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
}

/* Smooth transitions */
.v-card,
.v-btn,
.v-text-field {
  transition: all 0.3s ease;
}

/* Form styling improvements */
.v-text-field {
  margin-bottom: 8px;
}

.v-checkbox {
  margin-top: 8px;
}

/* Success state styling */
.v-avatar {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* Loading overlay */
.v-overlay {
  backdrop-filter: blur(4px);
}

/* Responsive adjustments */
@media (max-width: 600px) {
  .auth-card {
    margin: 16px;
    max-width: calc(100vw - 32px);
  }
  
  .theme-toggle {
    bottom: 16px;
    right: 16px;
  }
}
</style>