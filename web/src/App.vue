<script setup lang="ts">
// ThemePicker is parked here as a floating overlay for Phase 2. It moves
// into the persistent nav rail in Phase 4 — remove this block then.
import ThemePicker from './components/ThemePicker.vue'
</script>

<template>
  <RouterView />
  <div class="theme-picker-float">
    <ThemePicker />
  </div>
</template>

<style scoped>
.theme-picker-float {
  position: fixed;
  right: var(--space-6);
  bottom: var(--space-6);
  z-index: 50;
  box-shadow: var(--shadow-md);
  border-radius: var(--radius-md);
}
</style>
