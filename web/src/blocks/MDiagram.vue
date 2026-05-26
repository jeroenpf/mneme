<script setup lang="ts">
defineProps<{ id?: string; content?: string }>()
</script>

<template>
  <pre class="bg-elevated border border-border rounded-md p-4 mn-mono-sm">{{ content }}</pre>
</template>
