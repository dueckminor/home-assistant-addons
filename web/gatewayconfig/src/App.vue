<template>
  <v-app>
    <v-main>
      <Index />
    </v-main>
  </v-app>
</template>

<script>
import Index from './pages/Index.vue'

export default {
  name: 'App',
  components: {
    Index
  }
}
</script>