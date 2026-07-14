<script setup lang="ts">
// The persistent nav rail (with the ThemePicker) wraps every routed view.
import AppShell from './components/AppShell.vue'
</script>

<template>
  <AppShell>
    <RouterView />
  </AppShell>
</template>
