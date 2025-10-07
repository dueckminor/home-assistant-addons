<template>
  <div class="text-center pa-8">
    <v-icon size="64" color="success" class="mb-4">mdi-routes</v-icon>
    <h2 class="text-h5 mb-4">Route Management</h2>
    <p class="text-body-1 mb-4">
      Configure routing rules, domains, and proxy targets for your services.
    </p>
    <v-chip color="primary" class="ma-1">
      <v-icon start>mdi-web</v-icon>
      Domain Mapping
    </v-chip>
    <v-chip color="success" class="ma-1">
      <v-icon start>mdi-swap-horizontal</v-icon>
      Reverse Proxy
    </v-chip>
    <v-chip color="warning" class="ma-1">
      <v-icon start>mdi-security</v-icon>
      Authentication
    </v-chip>
  </div>
</template>

<script>
export default {
  name: 'RoutesTab',
  props: {
    routeConfig: {
      type: Object,
      required: true
    }
  },
  emits: ['save-config']
}
</script>