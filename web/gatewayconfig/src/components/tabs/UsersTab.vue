<template>
  <div class="pa-4">
    <!-- Header -->
    <div class="d-flex justify-space-between align-center mb-6">
      <div>
        <h2 class="text-h5 mb-2">Users & Groups</h2>
        <p class="text-body-2 text-medium-emphasis">
          Manage user accounts and access groups for OAuth authentication
        </p>
      </div>
    </div>

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
      <!-- Users Section -->
      <v-col cols="12" lg="6">
        <v-card>
          <v-card-title class="d-flex justify-space-between align-center">
            <span>
              <v-icon class="me-2">mdi-account</v-icon>
              Users
            </span>
            <v-btn
              color="primary"
              size="small"
              prepend-icon="mdi-plus"
              @click="openAddUserDialog"
            >
              Add User
            </v-btn>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <!-- Users List -->
            <div v-if="users.length > 0">
              <v-list>
                <v-list-item
                  v-for="user in users"
                  :key="user.guid"
                  class="user-item"
                >
                  <template v-slot:prepend>
                    <v-avatar size="32" color="primary">
                      <span class="text-white font-weight-bold">
                        {{ getUserInitials(user.name) }}
                      </span>
                    </v-avatar>
                  </template>
                  
                  <v-list-item-title>{{ user.name }}</v-list-item-title>
                  <v-list-item-subtitle>
                    {{ user.mail }}
                    <div class="mt-1">
                      <v-chip
                        v-for="groupName in user.groups"
                        :key="groupName"
                        size="x-small"
                        :color="getGroupColor(groupName)"
                        class="me-1"
                      >
                        {{ groupName }}
                      </v-chip>
                    </div>
                  </v-list-item-subtitle>
                  
                  <template v-slot:append>
                    <v-btn
                      icon="mdi-pencil"
                      variant="text"
                      size="small"
                      @click="openEditUserDialog(user)"
                      title="Edit user"
                    ></v-btn>
                    <v-btn
                      icon="mdi-delete"
                      variant="text"
                      size="small"
                      color="error"
                      @click="deleteUser(user.guid)"
                      title="Delete user"
                    ></v-btn>
                  </template>
                </v-list-item>
              </v-list>
            </div>
            
            <!-- Empty Users State -->
            <div v-else class="text-center pa-8">
              <v-icon size="48" color="grey" class="mb-2">mdi-account-outline</v-icon>
              <p class="text-body-2 text-medium-emphasis mb-4">
                No users configured
              </p>
              <v-btn
                color="primary"
                prepend-icon="mdi-plus"
                @click="openAddUserDialog"
              >
                Add First User
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>

      <!-- Groups Section -->
      <v-col cols="12" lg="6">
        <v-card>
          <v-card-title class="d-flex justify-space-between align-center">
            <span>
              <v-icon class="me-2">mdi-account-group</v-icon>
              Groups
            </span>
            <v-btn
              color="primary"
              size="small"
              prepend-icon="mdi-plus"
              @click="openAddGroupDialog"
            >
              Add Group
            </v-btn>
          </v-card-title>
          
          <v-card-text class="pa-0">
            <!-- Groups List -->
            <div v-if="groups.length > 0">
              <v-list>
                <v-list-item
                  v-for="group in groups"
                  :key="group.guid"
                  class="group-item"
                >
                  <template v-slot:prepend>
                    <v-icon color="primary">mdi-account-group</v-icon>
                  </template>
                  
                  <v-list-item-title>{{ group.name }}</v-list-item-title>
                  <v-list-item-subtitle>
                    {{ getUserCountInGroup(group.name) }} member(s)
                  </v-list-item-subtitle>
                  
                  <template v-slot:append>
                    <v-btn
                      icon="mdi-pencil"
                      variant="text"
                      size="small"
                      @click="openEditGroupDialog(group)"
                      title="Edit group"
                    ></v-btn>
                    <v-btn
                      icon="mdi-delete"
                      variant="text"
                      size="small"
                      color="error"
                      @click="deleteGroup(group.guid)"
                      title="Delete group"
                      :disabled="group.name === 'admin'"
                    ></v-btn>
                  </template>
                </v-list-item>
              </v-list>
            </div>
            
            <!-- Empty Groups State -->
            <div v-else class="text-center pa-8">
              <v-icon size="48" color="grey" class="mb-2">mdi-account-group-outline</v-icon>
              <p class="text-body-2 text-medium-emphasis mb-4">
                No groups configured
              </p>
              <v-btn
                color="primary"
                prepend-icon="mdi-plus"
                @click="openAddGroupDialog"
              >
                Add First Group
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <!-- Add/Edit Group Dialog -->
    <v-dialog v-model="groupDialog" max-width="500px">
      <v-card>
        <v-card-title>
          <v-icon class="me-2">mdi-account-group</v-icon>
          {{ editingGroup ? 'Edit Group' : 'Add Group' }}
        </v-card-title>
        
        <v-card-text>
          <v-form ref="groupForm" v-model="groupFormValid">
            <v-text-field
              v-model="groupFormData.name"
              label="Group Name"
              variant="outlined"
              prepend-inner-icon="mdi-account-group"
              :rules="groupNameRules"
              required
            ></v-text-field>
          </v-form>
        </v-card-text>
        
        <v-card-actions class="justify-end">
          <v-btn
            variant="text"
            @click="closeGroupDialog"
          >
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            @click="saveGroup"
            :disabled="!groupFormValid"
            :loading="saving"
          >
            {{ editingGroup ? 'Update' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Add/Edit User Dialog -->
    <v-dialog v-model="userDialog" max-width="600px">
      <v-card>
        <v-card-title>
          <v-icon class="me-2">mdi-account</v-icon>
          {{ editingUser ? 'Edit User' : 'Add User' }}
        </v-card-title>
        
        <v-card-text>
          <v-form ref="userForm" v-model="userFormValid">
            <v-text-field
              v-model="userFormData.name"
              label="Username"
              variant="outlined"
              prepend-inner-icon="mdi-account"
              :rules="userNameRules"
              required
            ></v-text-field>
            
            <v-text-field
              v-model="userFormData.mail"
              label="Email Address"
              variant="outlined"
              prepend-inner-icon="mdi-email"
              :rules="emailRules"
              required
            ></v-text-field>
            
            <v-select
              v-model="userFormData.groups"
              :items="groups.map(g => g.name)"
              label="Groups"
              variant="outlined"
              prepend-inner-icon="mdi-account-group"
              multiple
              chips
              closable-chips
              :rules="groupsRules"
              required
            >
              <template v-slot:chip="{ props, item }">
                <v-chip
                  v-bind="props"
                  :color="getGroupColor(item.raw)"
                >
                  {{ item.raw }}
                </v-chip>
              </template>
            </v-select>
          </v-form>
        </v-card-text>
        
        <v-card-actions class="justify-end">
          <v-btn
            variant="text"
            @click="closeUserDialog"
          >
            Cancel
          </v-btn>
          <v-btn
            v-if="editingUser"
            color="warning"
            variant="tonal"
            prepend-icon="mdi-key-change"
            @click="resetPassword"
            :loading="resettingPassword"
            class="me-2"
          >
            Reset Password
          </v-btn>
          <v-btn
            color="primary"
            @click="saveUser"
            :disabled="!userFormValid"
            :loading="saving"
          >
            {{ editingUser ? 'Update' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script>
import { apiRequest, apiGet } from '../../utils/api.js'

export default {
  name: 'UsersTab',
  data() {
    return {
      loading: false,
      error: null,
      saving: false,
      resettingPassword: false,
      users: [],
      groups: [],
      
      // Group Dialog
      groupDialog: false,
      editingGroup: null,
      groupFormValid: false,
      groupFormData: {
        name: ''
      },
      
      // User Dialog
      userDialog: false,
      editingUser: null,
      userFormValid: false,
      userFormData: {
        name: '',
        mail: '',
        groups: []
      },
      
      // Validation Rules
      groupNameRules: [
        v => !!v || 'Group name is required',
        v => /^[a-zA-Z0-9_-]+$/.test(v) || 'Group name can only contain letters, numbers, hyphens and underscores',
        v => v.length >= 2 || 'Group name must be at least 2 characters',
        v => !this.groups.some(g => g.name === v && (!this.editingGroup || g.guid !== this.editingGroup.guid)) || 'Group name already exists'
      ],
      
      userNameRules: [
        v => !!v || 'Username is required',
        v => /^[a-zA-Z0-9_-]+$/.test(v) || 'Username can only contain letters, numbers, hyphens and underscores',
        v => v.length >= 2 || 'Username must be at least 2 characters',
        v => !this.users.some(u => u.name === v && (!this.editingUser || u.guid !== this.editingUser.guid)) || 'Username already exists'
      ],
      
      emailRules: [
        v => !!v || 'Email is required',
        v => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) || 'Email must be valid'
      ],
      
      groupsRules: [
        v => v && v.length > 0 || 'At least one group must be selected'
      ]
    }
  },
  
  mounted() {
    this.loadData()
  },
  
  methods: {
    async loadData() {
      this.loading = true
      this.error = null
      
      try {
        await Promise.all([
          this.loadGroups(),
          this.loadUsers()
        ])
      } catch (error) {
        this.error = `Failed to load data: ${error.message}`
        console.error('Error loading users and groups:', error)
      } finally {
        this.loading = false
      }
    },
    
    async loadGroups() {
      try {
        const response = await apiGet('groups')
        this.groups = response.groups || []
        console.log('Loaded groups:', this.groups)
      } catch (error) {
        console.error('Error loading groups:', error)
        throw error
      }
    },
    
    async loadUsers() {
      try {
        const response = await apiGet('users')
        this.users = response.users || []
        console.log('Loaded users:', this.users)
      } catch (error) {
        console.error('Error loading users:', error)
        throw error
      }
    },
    
    // Group Management
    openAddGroupDialog() {
      this.editingGroup = null
      this.groupFormData = {
        name: ''
      }
      this.groupDialog = true
    },
    
    openEditGroupDialog(group) {
      this.editingGroup = group
      this.groupFormData = {
        name: group.name
      }
      this.groupDialog = true
    },
    
    closeGroupDialog() {
      this.groupDialog = false
      this.editingGroup = null
      this.groupFormData = {
        name: ''
      }
    },
    
    async saveGroup() {
      if (!this.groupFormValid) return
      
      this.saving = true
      try {
        if (this.editingGroup) {
          // Update existing group
          const response = await apiRequest(`groups/${this.editingGroup.guid}`, {
            method: 'PUT',
            body: JSON.stringify({
              name: this.groupFormData.name
            })
          })
          
          if (!response.ok) {
            throw new Error(`Failed to update group: ${response.status} ${response.statusText}`)
          }
        } else {
          // Create new group
          const response = await apiRequest('groups', {
            method: 'POST',
            body: JSON.stringify({
              name: this.groupFormData.name
            })
          })
          
          if (!response.ok) {
            throw new Error(`Failed to create group: ${response.status} ${response.statusText}`)
          }
        }
        
        this.closeGroupDialog()
        await this.loadGroups()
        
      } catch (error) {
        this.error = `Failed to save group: ${error.message}`
        console.error('Error saving group:', error)
      } finally {
        this.saving = false
      }
    },
    
    async deleteGroup(groupGuid) {
      if (!confirm('Are you sure you want to delete this group? Users in this group will lose access.')) {
        return
      }
      
      try {
        const response = await apiRequest(`groups/${groupGuid}`, {
          method: 'DELETE'
        })
        
        if (!response.ok) {
          throw new Error(`Failed to delete group: ${response.status} ${response.statusText}`)
        }
        
        await this.loadData() // Reload both users and groups since users might be affected
        
      } catch (error) {
        this.error = `Failed to delete group: ${error.message}`
        console.error('Error deleting group:', error)
      }
    },
    
    // User Management
    openAddUserDialog() {
      this.editingUser = null
      this.userFormData = {
        name: '',
        mail: '',
        groups: []
      }
      this.userDialog = true
    },
    
    openEditUserDialog(user) {
      this.editingUser = user
      this.userFormData = {
        name: user.name,
        mail: user.mail,
        groups: [...user.groups]
      }
      this.userDialog = true
    },
    
    closeUserDialog() {
      this.userDialog = false
      this.editingUser = null
      this.userFormData = {
        name: '',
        mail: '',
        groups: []
      }
    },
    
    async saveUser() {
      if (!this.userFormValid) return
      
      this.saving = true
      try {
        if (this.editingUser) {
          // Update existing user
          const response = await apiRequest(`users/${this.editingUser.guid}`, {
            method: 'PUT',
            body: JSON.stringify({
              name: this.userFormData.name,
              mail: this.userFormData.mail,
              groups: this.userFormData.groups
            })
          })
          
          if (!response.ok) {
            throw new Error(`Failed to update user: ${response.status} ${response.statusText}`)
          }
        } else {
          // Create new user
          const response = await apiRequest('users', {
            method: 'POST',
            body: JSON.stringify({
              name: this.userFormData.name,
              mail: this.userFormData.mail,
              groups: this.userFormData.groups
            })
          })
          
          if (!response.ok) {
            throw new Error(`Failed to create user: ${response.status} ${response.statusText}`)
          }
        }
        
        this.closeUserDialog()
        await this.loadUsers()
        
      } catch (error) {
        this.error = `Failed to save user: ${error.message}`
        console.error('Error saving user:', error)
      } finally {
        this.saving = false
      }
    },
    
    async deleteUser(userGuid) {
      if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
        return
      }
      
      try {
        const response = await apiRequest(`users/${userGuid}`, {
          method: 'DELETE'
        })
        
        if (!response.ok) {
          throw new Error(`Failed to delete user: ${response.status} ${response.statusText}`)
        }
        
        await this.loadUsers()
        
      } catch (error) {
        this.error = `Failed to delete user: ${error.message}`
        console.error('Error deleting user:', error)
      }
    },
    
    async resetPassword() {
      if (!this.editingUser) {
        return
      }
      
      if (!confirm(`Are you sure you want to reset the password for ${this.editingUser.name}? A password reset email will be sent.`)) {
        return
      }
      
      this.resettingPassword = true
      try {
        const response = await apiRequest(`users/${this.editingUser.guid}/password_reset`, {
          method: 'POST'
        })
        
        if (!response.ok) {
          throw new Error(`Failed to reset password: ${response.status} ${response.statusText}`)
        }
        
        // Show success message
        this.error = null
        // You might want to show a success snackbar here instead
        alert(`Password reset email has been sent to ${this.editingUser.mail}`)
        
      } catch (error) {
        this.error = `Failed to reset password: ${error.message}`
        console.error('Error resetting password:', error)
      } finally {
        this.resettingPassword = false
      }
    },
    
    // Helper Methods
    getUserInitials(name) {
      return name
        .split(' ')
        .map(word => word.charAt(0))
        .join('')
        .substring(0, 2)
        .toUpperCase()
    },
    
    getUserCountInGroup(groupName) {
      return this.users.filter(user => user.groups.includes(groupName)).length
    },
    
    getGroupColor(groupName) {
      const colors = {
        'admin': 'red',
        'user': 'blue',
        'guest': 'grey'
      }
      return colors[groupName] || 'primary'
    }
  }
}
</script>

<style scoped>
.group-item:hover,
.user-item:hover {
  background-color: rgba(0, 0, 0, 0.04);
}
</style>