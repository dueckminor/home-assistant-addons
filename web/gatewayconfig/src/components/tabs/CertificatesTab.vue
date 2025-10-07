<template>
  <div class="text-center pa-8">
    <v-icon size="64" color="green" class="mb-4">mdi-certificate</v-icon>
    <h2 class="text-h5 mb-4">SSL Certificates</h2>
    <p class="text-body-1 mb-4">
      Configure Let's Encrypt certificates, PKI management, and TLS settings.
    </p>
    <v-chip color="green" class="ma-1">
      <v-icon start>mdi-auto-fix</v-icon>
      Auto-renewal
    </v-chip>
    <v-chip color="teal" class="ma-1">
      <v-icon start>mdi-security</v-icon>
      Wildcard Support
    </v-chip>
  </div>
</template>

<script>
export default {
  name: 'CertificatesTab',
  props: {
    certificateConfig: {
      type: Object,
      required: true
    }
  },
  emits: ['save-config']
}
</script>