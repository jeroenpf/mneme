<script setup lang="ts">
defineProps<{ id?: string; variant?: string; content?: string }>()
</script>

<template>
  <aside class="border border-border rounded-md p-4 bg-elevated mn-body-sm">
    <strong class="mn-label">{{ variant ?? 'note' }}</strong>
    <div>{{ content }}</div>
  </aside>
</template>
