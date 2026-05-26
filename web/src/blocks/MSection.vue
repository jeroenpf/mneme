<script setup lang="ts">
import BlockRenderer from './BlockRenderer.vue'
defineProps<{ id?: string; title?: string; children?: unknown[] }>()
</script>

<template>
  <section class="space-y-3">
    <h2 v-if="title" class="mn-h2">{{ title }}</h2>
    <BlockRenderer :blocks="(children ?? []) as Array<Record<string, unknown>>" />
  </section>
</template>
