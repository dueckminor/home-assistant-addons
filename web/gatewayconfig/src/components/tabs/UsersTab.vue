<template>
  <div class="text-center pa-8">
    <v-icon size="64" color="orange" class="mb-4">mdi-account-group</v-icon>
    <h2 class="text-h5 mb-4">User Management</h2>
    <p class="text-body-1 mb-4">
      Manage authentication, user accounts, and access permissions.
    </p>
    <v-chip color="purple" class="ma-1">
      <v-icon start>mdi-key</v-icon>
      OAuth2
    </v-chip>
    <v-chip color="indigo" class="ma-1">
      <v-icon start>mdi-account-lock</v-icon>
      Access Control
    </v-chip>
  </div>
</template>

<script>
export default {
  name: 'UsersTab',
  props: {
    userConfig: {
      type: Object,
      required: true
    }
  },
  emits: ['save-config']
}
</script>