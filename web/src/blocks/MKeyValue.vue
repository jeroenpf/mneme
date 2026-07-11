<script setup lang="ts">
import { renderInline } from '@/lib/markdown'

defineProps<{ id?: string; title?: string; data?: Record<string, string> }>()
</script>

<template>
  <div v-if="data && Object.keys(data).length">
    <h4 v-if="title" class="mn-label mb-2">{{ title }}</h4>
    <dl class="kv">
      <template v-for="(value, key) in data" :key="key">
        <dt class="mn-label">{{ key }}</dt>
        <dd class="mn-body-sm mn-md" v-html="renderInline(value)" />
      </template>
    </dl>
  </div>
</template>

<style scoped>
.kv {
  display: grid;
  grid-template-columns: minmax(120px, auto) 1fr;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  overflow: hidden;
}
.kv dt,
.kv dd {
  padding: var(--space-2) var(--space-4);
  border-top: 1px solid var(--border-soft);
}
.kv dt:first-of-type,
.kv dd:first-of-type {
  border-top: none;
}
.kv dt {
  align-self: center;
}
</style>
