<script setup lang="ts">
defineProps<{ id?: string; language?: string; content?: string }>()
</script>

<template>
  <pre class="bg-elevated border border-border rounded-md p-4 mn-mono-sm overflow-auto"><code>{{ content }}</code></pre>
</template>
