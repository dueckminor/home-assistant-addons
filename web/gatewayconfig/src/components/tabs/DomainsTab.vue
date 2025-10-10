<template>
  <v-row>
    <v-col cols="12">
      <div class="text-center mb-6">
        <v-icon size="48" color="purple" class="mb-3">mdi-domain</v-icon>
        <h2 class="text-h5 mb-2">Domain Management</h2>
        <p class="text-body-2 text-medium-emphasis">
          Configure which domains this gateway's DNS server should handle
        </p>
      </div>
    </v-col>
  </v-row>

  <!-- Managed Domains -->
  <v-row>
    <v-col cols="12">
      <v-card>
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="purple">mdi-domain</v-icon>
          Managed Domains
        </v-card-title>
        <v-card-subtitle>
          Configure which domains this gateway's DNS server should handle
        </v-card-subtitle>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="8">
              <v-text-field
                v-model="newDomain"
                label="Add Domain"
                variant="outlined"
                prepend-inner-icon="mdi-web"
                placeholder="example.com"
                hint="Enter a domain that should be handled by this DNS server"
                persistent-hint
                @keyup.enter="addDomain"
              >
                <template v-slot:append-inner>
                  <v-btn
                    icon="mdi-plus"
                    variant="text"
                    size="small"
                    @click="addDomain"
                    :disabled="!newDomain.trim()"
                  ></v-btn>
                </template>
              </v-text-field>
            </v-col>
            <v-col cols="12" md="4">
              <v-btn
                color="primary"
                variant="outlined"
                prepend-icon="mdi-plus"
                @click="addDomain"
                :disabled="!newDomain.trim()"
                block
              >
                Add Domain
              </v-btn>
            </v-col>
          </v-row>

          <!-- Domain List -->
          <v-row v-if="domains && domains.length > 0" class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <h4 class="text-subtitle-1 mb-3">
                <v-icon class="me-2" size="small">mdi-format-list-bulleted</v-icon>
                Configured Domains ({{ domains.length }})
              </h4>
              <v-row>
                <v-col 
                  v-for="(domain, index) in domains" 
                  :key="index"
                  cols="12" 
                  md="6" 
                  lg="4"
                >
                  <v-card variant="outlined" class="domain-card">
                    <v-card-text class="d-flex align-center pa-3">
                      <v-icon class="me-3" color="success">mdi-check-circle</v-icon>
                      <span class="flex-grow-1 text-body-2">{{ domain }}</span>
                      <v-btn
                        icon="mdi-delete"
                        variant="text"
                        size="small"
                        color="error"
                        @click="removeDomain(index)"
                      ></v-btn>
                    </v-card-text>
                  </v-card>
                </v-col>
              </v-row>
            </v-col>
          </v-row>

          <!-- Empty State -->
          <v-row v-else class="mt-2">
            <v-col cols="12">
              <v-divider class="mb-4"></v-divider>
              <div class="text-center pa-4">
                <v-icon size="48" color="grey" class="mb-2">mdi-domain-off</v-icon>
                <p class="text-body-2 text-medium-emphasis">
                  No domains configured. Add domains that should be handled by this DNS server.
                </p>
              </div>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>

  <!-- DNS Server Status -->
  <v-row class="mt-4">
    <v-col cols="12">
      <v-card color="grey-lighten-5">
        <v-card-title class="text-h6 d-flex align-center">
          <v-icon class="me-2" color="success">mdi-server</v-icon>
          DNS Server Status
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="3">
              <v-chip color="success" class="ma-1">
                <v-icon start>mdi-port</v-icon>
                Port: 53 (UDP/TCP)
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="info" class="ma-1">
                <v-icon start>mdi-earth</v-icon>
                External IP Detection
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="purple" class="ma-1">
                <v-icon start>mdi-domain</v-icon>
                {{ (domains && domains.length) || 0 }} Domains
              </v-chip>
            </v-col>
            <v-col cols="12" md="3">
              <v-chip color="primary" class="ma-1">
                <v-icon start>mdi-check-circle</v-icon>
                Running
              </v-chip>
            </v-col>
          </v-row>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>

<script>
export default {
  name: 'DomainsTab',
  props: {
    domains: {
      type: Array,
      required: true
    }
  },
  emits: ['save-config'],
  data() {
    return {
      newDomain: ''
    }
  },
  methods: {
    addDomain() {
      const domain = this.newDomain.trim().toLowerCase();
      if (!domain) return;
      
      // Basic domain validation
      if (!/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(domain)) {
        console.error('Invalid domain format:', domain);
        // TODO: Show error message to user
        return;
      }
      
      // Check for duplicates
      if (this.domains.includes(domain)) {
        console.warn('Domain already exists:', domain);
        // TODO: Show warning message to user
        return;
      }
      
      // Add domain
      this.domains.push(domain);
      this.newDomain = '';
      console.log('Domain added:', domain);
      
      // Emit save event
      this.$emit('save-config');
    },
    removeDomain(index) {
      const domain = this.domains[index];
      this.domains.splice(index, 1);
      console.log('Domain removed:', domain);
      
      // Emit save event
      this.$emit('save-config');
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