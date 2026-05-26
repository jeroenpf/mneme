<script setup lang="ts">
defineProps<{ id: string }>()
</script>

<template>
  <section class="p-6">
    <h1 class="mn-display">Document</h1>
    <p class="mn-body-sm">Viewer for <code class="mn-code-inline">{{ id }}</code> lands in Phase 1.7.</p>
  </section>
</template>
