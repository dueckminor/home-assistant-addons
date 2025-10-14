<template>
  <v-row>
    <v-col cols="12">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="primary">mdi-dns</v-icon>
          DNS Configuration
        </v-card-title>
        <v-card-subtitle>
          Configure DNS server settings and external IP detection
        </v-card-subtitle>
      </v-card>
    </v-col>
  </v-row>

  <v-row>
    <!-- IPv4 Configuration -->
    <v-col cols="12" md="6">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="primary">mdi-ip-network</v-icon>
          IPv4 Configuration
        </v-card-title>
        <v-card-text>
          <v-select
            v-model="dnsConfig.ipv4.method"
            :items="ipDetectionMethods"
            label="Detection Method"
            variant="outlined"
            prepend-inner-icon="mdi-cog"
            class="mb-4"
          ></v-select>
          
          <v-text-field
            v-model="dnsConfig.ipv4.source"
            label="IPv4 Source Address"
            variant="outlined"
            prepend-inner-icon="mdi-web"
            placeholder="example.myfritz.net"
            hint="The address to query for your external IPv4"
            persistent-hint
            class="mb-4"
          ></v-text-field>

          <v-text-field
            v-model="dnsConfig.ipv4.current"
            label="Current IPv4 Address"
            variant="outlined"
            prepend-inner-icon="mdi-ip"
            readonly
            placeholder="Detecting..."
            :loading="ipv4Loading"
            class="mb-2"
          >
            <template v-slot:append-inner>
              <v-btn
                v-if="showIpv4Revert"
                icon="mdi-undo"
                variant="text"
                size="small"
                color="warning"
                @click="revertIPv4"
                title="Revert to last working configuration"
                class="me-1"
              ></v-btn>
              <v-btn
                icon="mdi-refresh"
                variant="text"
                size="small"
                @click="refreshIPv4"
                :loading="ipv4Loading"
              ></v-btn>
            </template>
          </v-text-field>
          
          <div class="text-caption text-medium-emphasis">
            Last updated: {{ dnsConfig.ipv4.lastUpdate || 'Never' }}
          </div>
        </v-card-text>
      </v-card>
    </v-col>

    <!-- IPv6 Configuration -->
    <v-col cols="12" md="6">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="secondary">mdi-ip-network-outline</v-icon>
          IPv6 Configuration
        </v-card-title>
        <v-card-text>
          <v-select
            v-model="dnsConfig.ipv6.method"
            :items="ipDetectionMethods"
            label="Detection Method"
            variant="outlined"
            prepend-inner-icon="mdi-cog"
            class="mb-4"
          ></v-select>
          
          <v-text-field
            v-model="dnsConfig.ipv6.source"
            label="IPv6 Source Address"
            variant="outlined"
            prepend-inner-icon="mdi-web"
            placeholder="homeassistant.local"
            hint="The address to query for your external IPv6"
            persistent-hint
            class="mb-4"
          ></v-text-field>

          <v-text-field
            v-model="dnsConfig.ipv6.current"
            label="Current IPv6 Address"
            variant="outlined"
            prepend-inner-icon="mdi-ip"
            readonly
            placeholder="Detecting..."
            :loading="ipv6Loading"
            class="mb-2"
          >
            <template v-slot:append-inner>
              <v-btn
                v-if="showIpv6Revert"
                icon="mdi-undo"
                variant="text"
                size="small"
                color="warning"
                @click="revertIPv6"
                title="Revert to last working configuration"
                class="me-1"
              ></v-btn>
              <v-btn
                icon="mdi-refresh"
                variant="text"
                size="small"
                @click="refreshIPv6"
                :loading="ipv6Loading"
              ></v-btn>
            </template>
          </v-text-field>
          
          <div class="text-caption text-medium-emphasis">
            Last updated: {{ dnsConfig.ipv6.lastUpdate || 'Never' }}
          </div>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>

<script>
export default {
  name: 'DnsTab',
  props: {
    dnsConfig: {
      type: Object,
      required: true
    },
    ipDetectionMethods: {
      type: Array,
      required: true
    },
    ipv4Loading: {
      type: Boolean,
      default: false
    },
    ipv6Loading: {
      type: Boolean,
      default: false
    },
    showIpv4Revert: {
      type: Boolean,
      default: false
    },
    showIpv6Revert: {
      type: Boolean,
      default: false
    }
  },
  emits: ['refresh-ipv4', 'refresh-ipv6', 'revert-ipv4', 'revert-ipv6'],
  methods: {
    refreshIPv4() {
      this.$emit('refresh-ipv4');
    },
    refreshIPv6() {
      this.$emit('refresh-ipv6');
    },
    revertIPv4() {
      this.$emit('revert-ipv4');
    },
    revertIPv6() {
      this.$emit('revert-ipv6');
    }
  }
}
</script>

<style scoped>
.domain-card {
  transition: all 0.2s ease;
}

.domain-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}
</style>