<script setup lang="ts">
defineProps<{ id?: string; pairs?: Record<string, string> }>()
</script>

<template>
  <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
    <template v-for="(value, key) in pairs ?? {}" :key="key">
      <dt class="mn-label">{{ key }}</dt>
      <dd class="mn-body-sm">{{ value }}</dd>
    </template>
  </dl>
</template>
