<template>
  <v-container>
    <v-row justify="center">
      <v-col cols="12" md="8" lg="6">
        <!-- Header -->
        <v-card class="mb-6">
          <v-card-title class="text-h4 text-center pa-6">
            <v-icon class="me-2" color="primary">mdi-gateway</v-icon>
            Gateway Configuration
          </v-card-title>
        </v-card>

        <!-- Welcome Message -->
        <v-card>
          <v-card-text class="text-center pa-8">
            <v-icon size="64" color="success" class="mb-4">mdi-check-circle</v-icon>
            <h2 class="text-h5 mb-4">Configuration UI Ready</h2>
            <p class="text-body-1 mb-4">
              The gateway configuration interface is now running on port 8099.
            </p>
            <v-chip color="info" class="ma-1">
              <v-icon start>mdi-server</v-icon>
              Ingress Port: 8099
            </v-chip>
            <v-chip color="success" class="ma-1">
              <v-icon start>mdi-shield-check</v-icon>
              Internal Access Only
            </v-chip>
          </v-card-text>
        </v-card>

        <!-- Coming Soon -->
        <v-card class="mt-6">
          <v-card-text class="text-center pa-6">
            <v-icon size="48" color="warning" class="mb-3">mdi-construction</v-icon>
            <h3 class="text-h6 mb-3">Configuration Features Coming Soon</h3>
            <p class="text-body-2 text-medium-emphasis">
              Domain management, server routing, SSL configuration, and more...
            </p>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
export default {
  name: 'Index',
  data() {
    return {
      // Future configuration data will go here
    }
  },
  mounted() {
    console.log('Gateway Configuration UI loaded')
  }
}
</script>

<style scoped>
/* Custom styles for the configuration UI */
</style>