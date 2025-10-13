<template>
  <v-app :theme="isDark ? 'dark' : 'light'">
    <v-main class="auth-background">
      <v-container fluid class="fill-height">
        <v-row justify="center" align="center" class="fill-height">
          <v-col cols="12" sm="8" md="6" lg="4" xl="3">
            <!-- Authentication Success Card -->
            <v-card 
              v-if="authenticated || redirecting" 
              class="auth-card"
              elevation="8"
            >
              <v-card-title class="text-center pa-6">
                <div class="d-flex flex-column align-center">
                  <v-avatar v-if="redirecting" size="64" color="primary" class="mb-4">
                    <v-progress-circular indeterminate size="32" width="3"></v-progress-circular>
                  </v-avatar>
                  <v-avatar v-else size="64" color="success" class="mb-4">
                    <v-icon size="32">mdi-check-circle</v-icon>
                  </v-avatar>
                  <h2 v-if="redirecting" class="text-h5 mb-2">Redirecting...</h2>
                  <h2 v-else class="text-h5 mb-2">Welcome back!</h2>
                  <p v-if="!redirecting" class="text-body-1 text-medium-emphasis mb-0">
                    Logged in as <strong>{{ username }}</strong>
                  </p>
                </div>
              </v-card-title>

              <v-card-text v-if="!redirecting" class="pa-6 pt-0">
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
                    :disabled="loading || showForgotPasswordDialog"
                    autofocus
                    required
                    :name="showForgotPasswordDialog ? 'login-username-hidden' : 'username'"
                    :id="showForgotPasswordDialog ? 'login-username-hidden' : 'username'"
                    :autocomplete="showForgotPasswordDialog ? 'off' : 'username'"
                    :data-lpignore="showForgotPasswordDialog"
                    :data-bwignore="showForgotPasswordDialog"
                    :data-1p-ignore="showForgotPasswordDialog"
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
                    :disabled="loading || showForgotPasswordDialog"
                    required
                    :name="showForgotPasswordDialog ? 'login-password-hidden' : 'password'"
                    :id="showForgotPasswordDialog ? 'login-password-hidden' : 'password'"
                    :autocomplete="showForgotPasswordDialog ? 'off' : 'current-password'"
                    :data-lpignore="showForgotPasswordDialog"
                    :data-bwignore="showForgotPasswordDialog"
                    :data-1p-ignore="showForgotPasswordDialog"
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

                  <div class="text-center mt-4">
                    <v-btn
                      variant="text"
                      color="primary"
                      size="small"
                      @click="showForgotPasswordDialog = true"
                      :disabled="loading"
                    >
                      Forgot Password?
                    </v-btn>
                  </div>
                </v-form>
              </v-card-text>
            </v-card>

            <!-- Forgot Password Dialog -->
            <v-dialog v-model="showForgotPasswordDialog" max-width="500px">
              <v-card>
                <v-card-title class="d-flex align-center">
                  <v-icon class="me-2">mdi-key-variant</v-icon>
                  Reset Password
                </v-card-title>
                
                <v-card-text>
                  <p class="text-body-1 mb-4">
                    Enter your email address. If an account with that email exists, we'll send you instructions to reset your password.
                  </p>
                  
                  <v-form ref="forgotPasswordForm" v-model="forgotPasswordFormValid" @submit.prevent="sendResetEmail">
                    <v-text-field
                      v-model="resetEmail"
                      label="Email Address"
                      variant="outlined"
                      prepend-inner-icon="mdi-email"
                      type="email"
                      :rules="emailRules"
                      :disabled="sendingResetEmail"
                      required
                      autofocus
                    ></v-text-field>
                    
                    <!-- Success Message -->
                    <v-alert
                      v-if="resetEmailSent"
                      type="success"
                      variant="tonal"
                      class="mt-4"
                    >
                      <v-icon start>mdi-check-circle</v-icon>
                      If an account with that email exists, we've sent password reset instructions.
                    </v-alert>
                    
                    <!-- Error Message -->
                    <v-alert
                      v-if="resetEmailError"
                      type="error"
                      variant="tonal"
                      class="mt-4"
                      closable
                      @click:close="resetEmailError = ''"
                    >
                      <v-icon start>mdi-alert-circle</v-icon>
                      {{ resetEmailError }}
                    </v-alert>
                  </v-form>
                </v-card-text>
                
                <v-card-actions class="justify-end">
                  <v-btn
                    variant="text"
                    @click="closeForgotPasswordDialog"
                    :disabled="sendingResetEmail"
                  >
                    Cancel
                  </v-btn>
                  <v-btn
                    color="primary"
                    @click="sendResetEmail"
                    :loading="sendingResetEmail"
                    :disabled="!forgotPasswordFormValid"
                  >
                    Send Reset Email
                  </v-btn>
                </v-card-actions>
              </v-card>
            </v-dialog>

            <!-- Password Reset Dialog -->
            <v-dialog v-model="showPasswordResetDialog" max-width="500px" persistent>
              <v-card>
                <v-card-title class="d-flex align-center">
                  <v-icon class="me-2">mdi-lock-reset</v-icon>
                  Set New Password
                </v-card-title>
                
                <v-card-text>
                  <p class="text-body-1 mb-4">
                    Please enter your new password below.
                  </p>
                  
                  <v-form ref="passwordResetForm" v-model="passwordResetFormValid" @submit.prevent="submitNewPassword">
                    <v-text-field
                      v-model="newPassword"
                      label="New Password"
                      variant="outlined"
                      prepend-inner-icon="mdi-lock"
                      :type="showNewPassword ? 'text' : 'password'"
                      :append-inner-icon="showNewPassword ? 'mdi-eye' : 'mdi-eye-off'"
                      @click:append-inner="showNewPassword = !showNewPassword"
                      :rules="newPasswordRules"
                      :disabled="submittingNewPassword"
                      required
                      autofocus
                      autocomplete="new-password"
                    ></v-text-field>
                    
                    <v-text-field
                      v-model="confirmPassword"
                      label="Confirm New Password"
                      variant="outlined"
                      prepend-inner-icon="mdi-lock-check"
                      :type="showConfirmPassword ? 'text' : 'password'"
                      :append-inner-icon="showConfirmPassword ? 'mdi-eye' : 'mdi-eye-off'"
                      @click:append-inner="showConfirmPassword = !showConfirmPassword"
                      :rules="confirmPasswordRules"
                      :disabled="submittingNewPassword"
                      required
                      autocomplete="new-password"
                    ></v-text-field>
                    
                    <!-- Success Message -->
                    <v-alert
                      v-if="passwordResetSuccess"
                      type="success"
                      variant="tonal"
                      class="mt-4"
                    >
                      <v-icon start>mdi-check-circle</v-icon>
                      Password has been successfully updated. You can now sign in with your new password.
                    </v-alert>
                    
                    <!-- Error Message -->
                    <v-alert
                      v-if="passwordResetError"
                      type="error"
                      variant="tonal"
                      class="mt-4"
                      closable
                      @click:close="passwordResetError = ''"
                    >
                      <v-icon start>mdi-alert-circle</v-icon>
                      {{ passwordResetError }}
                    </v-alert>
                  </v-form>
                </v-card-text>
                
                <v-card-actions class="justify-end">
                  <v-btn
                    v-if="!passwordResetSuccess"
                    variant="text"
                    @click="closePasswordResetDialog"
                    :disabled="submittingNewPassword"
                  >
                    Cancel
                  </v-btn>
                  <v-btn
                    v-if="passwordResetSuccess"
                    color="primary"
                    @click="closePasswordResetDialog"
                  >
                    Continue to Sign In
                  </v-btn>
                  <v-btn
                    v-else
                    color="primary"
                    @click="submitNewPassword"
                    :loading="submittingNewPassword"
                    :disabled="!passwordResetFormValid"
                  >
                    Update Password
                  </v-btn>
                </v-card-actions>
              </v-card>
            </v-dialog>

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
    const redirecting = ref(false)
    const formValid = ref(false)
    const showPassword = ref(false)
    const rememberMe = ref(false)
    const isDark = ref(true)
    
    // Forgot password state
    const showForgotPasswordDialog = ref(false)
    const forgotPasswordFormValid = ref(false)
    const resetEmail = ref('')
    const sendingResetEmail = ref(false)
    const resetEmailSent = ref(false)
    const resetEmailError = ref('')
    
    // Password reset state
    const showPasswordResetDialog = ref(false)
    const passwordResetFormValid = ref(false)
    const newPassword = ref('')
    const confirmPassword = ref('')
    const showNewPassword = ref(false)
    const showConfirmPassword = ref(false)
    const submittingNewPassword = ref(false)
    const passwordResetSuccess = ref(false)
    const passwordResetError = ref('')
    const passwordResetToken = ref('')
    
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
    
    const emailRules = [
      v => !!v || 'Email is required',
      v => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) || 'Email must be valid'
    ]
    
    const newPasswordRules = [
      v => !!v || 'Password is required',
      v => v.length >= 8 || 'Password must be at least 8 characters',
      v => /(?=.*[a-z])/.test(v) || 'Password must contain at least one lowercase letter',
      v => /(?=.*[A-Z])/.test(v) || 'Password must contain at least one uppercase letter',
      v => /(?=.*\d)/.test(v) || 'Password must contain at least one number'
    ]
    
    const confirmPasswordRules = [
      v => !!v || 'Please confirm your password',
      v => v === newPassword.value || 'Passwords do not match'
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
          
          // Handle OAuth redirect if needed
          if (oauth.redirectURI && oauth.redirectURI !== '') {
            redirecting.value = true // Show redirecting state
            
            const redirectUrl = new URL('/oauth/authorize', window.location.origin)
            redirectUrl.searchParams.set('client_id', oauth.clientId)
            redirectUrl.searchParams.set('redirect_uri', oauth.redirectURI)
            redirectUrl.searchParams.set('response_type', oauth.responseType)
            
            // Redirect immediately - no delay needed
            window.location.href = redirectUrl.toString()
          } else {
            // Show welcome screen - no notification needed
            authenticated.value = true
            credentials.password = '' // Clear password for security
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
    
    const sendResetEmail = async () => {
      if (!forgotPasswordFormValid.value) return
      
      sendingResetEmail.value = true
      resetEmailError.value = ''
      resetEmailSent.value = false
      
      try {
        const response = await fetch('/send_reset_password_mail', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            mail: resetEmail.value
          })
        })
        
        if (response.ok || response.status === 202) {
          // Success - show confirmation message
          resetEmailSent.value = true
          showNotification('Password reset email sent successfully', 'success', 'mdi-email-check')
          
          // Auto-close dialog after a delay
          setTimeout(() => {
            closeForgotPasswordDialog()
          }, 3000)
        } else {
          // Handle different error responses
          let errorMsg = 'Failed to send reset email'
          
          if (response.status === 400) {
            errorMsg = 'Please enter a valid email address'
          } else if (response.status === 429) {
            errorMsg = 'Too many requests. Please try again later.'
          } else if (response.status >= 500) {
            errorMsg = 'Server error. Please try again later.'
          }
          
          resetEmailError.value = errorMsg
        }
      } catch (error) {
        console.error('Reset email error:', error)
        resetEmailError.value = 'Connection failed. Please check your network and try again.'
      } finally {
        sendingResetEmail.value = false
      }
    }
    
    const closeForgotPasswordDialog = () => {
      showForgotPasswordDialog.value = false
      resetEmail.value = ''
      resetEmailSent.value = false
      resetEmailError.value = ''
      forgotPasswordFormValid.value = false
    }
    
    const submitNewPassword = async () => {
      if (!passwordResetFormValid.value || !passwordResetToken.value) return
      
      submittingNewPassword.value = true
      passwordResetError.value = ''
      passwordResetSuccess.value = false
      
      try {
        const response = await fetch('/reset_password', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            token: passwordResetToken.value,
            password: newPassword.value
          })
        })
        
        if (response.ok) {
          // Password reset successful
          passwordResetSuccess.value = true
          showNotification('Password updated successfully!', 'success', 'mdi-check-circle')
          
          // Clear password fields for security
          newPassword.value = ''
          confirmPassword.value = ''
        } else {
          // Handle different error responses
          let errorMsg = 'Failed to update password'
          
          if (response.status === 400) {
            errorMsg = 'Invalid or expired reset token'
          } else if (response.status === 422) {
            errorMsg = 'Password does not meet requirements'
          } else if (response.status >= 500) {
            errorMsg = 'Server error. Please try again later.'
          }
          
          passwordResetError.value = errorMsg
        }
      } catch (error) {
        console.error('Password reset error:', error)
        passwordResetError.value = 'Connection failed. Please check your network and try again.'
      } finally {
        submittingNewPassword.value = false
      }
    }
    
    const closePasswordResetDialog = () => {
      showPasswordResetDialog.value = false
      newPassword.value = ''
      confirmPassword.value = ''
      passwordResetSuccess.value = false
      passwordResetError.value = ''
      passwordResetFormValid.value = false
      showNewPassword.value = false
      showConfirmPassword.value = false
      passwordResetToken.value = ''
      
      // Remove the token from URL
      const url = new URL(window.location.href)
      url.searchParams.delete('password_reset')
      window.history.replaceState({}, '', url.toString())
    }
    
    const checkForPasswordReset = () => {
      const url = new URL(window.location.href)
      const token = url.searchParams.get('password_reset')
      
      if (token) {
        passwordResetToken.value = token
        showPasswordResetDialog.value = true
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
      checkForPasswordReset()
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
      redirecting,
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
      
      // Forgot password state
      showForgotPasswordDialog,
      forgotPasswordFormValid,
      resetEmail,
      sendingResetEmail,
      resetEmailSent,
      resetEmailError,
      
      // Password reset state
      showPasswordResetDialog,
      passwordResetFormValid,
      newPassword,
      confirmPassword,
      showNewPassword,
      showConfirmPassword,
      submittingNewPassword,
      passwordResetSuccess,
      passwordResetError,
      
      // Computed
      username: computed(() => credentials.username),
      password: computed({
        get: () => credentials.password,
        set: (value) => { credentials.password = value }
      }),
      
      // Validation
      usernameRules,
      passwordRules,
      emailRules,
      newPasswordRules,
      confirmPasswordRules,
      
      // Methods
      login,
      logout,
      focusPassword,
      toggleTheme,
      showNotification,
      handleUsernameInput,
      handlePasswordInput,
      sendResetEmail,
      closeForgotPasswordDialog,
      submitNewPassword,
      closePasswordResetDialog
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