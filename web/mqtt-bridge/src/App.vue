<template>
  <v-app>
    <!-- Header -->
    <v-app-bar color="primary" dark elevation="2">
      <v-icon class="me-3">mdi-network-outline</v-icon>
      <v-toolbar-title>MQTT Bridge Configuration</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-chip color="success" variant="outlined">
        <v-icon start>mdi-shield-check</v-icon>
        Secure Access
      </v-chip>
    </v-app-bar>

    <!-- Main Content -->
    <v-main>
      <v-container fluid class="pa-4">
        <v-row>
          <v-col cols="12">
            <!-- Navigation Tabs -->
            <v-card>
              <v-tabs v-model="activeTab" bg-color="primary" slider-color="white">
                <v-tab value="mqtt">
                  <v-icon start>mdi-network-outline</v-icon>
                  MQTT
                </v-tab>
                <v-tab value="esp-logs">
                  <v-icon start>mdi-console-line</v-icon>
                  ESP Logs
                </v-tab>
              </v-tabs>

              <v-card-text class="pa-4">
                <v-tabs-window v-model="activeTab" class="w-100">
                  <!-- MQTT Tab -->
                  <v-tabs-window-item value="mqtt" class="w-100">
                    <MqttTab />
                  </v-tabs-window-item>
                  
                  <!-- ESP Logs Tab -->
                  <v-tabs-window-item value="esp-logs" class="w-100">
                    <EspLogsTab />
                  </v-tabs-window-item>
                </v-tabs-window>
              </v-card-text>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </v-main>
  </v-app>
</template>

<script>
import MqttTab from './components/tabs/MqttTab.vue'
import EspLogsTab from './components/tabs/EspLogsTab.vue'

export default {
  name: 'App',
  components: {
    MqttTab,
    EspLogsTab
  },
  data() {
    return {
      activeTab: 'mqtt'
    }
  }
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>