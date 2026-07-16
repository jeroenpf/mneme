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
/* Two-tone: a shaded key column separated by a divider from a lighter value
   column. The tone contrast makes rows scannable, and the light value column
   keeps the block legible even when it sits inside a surface-colored card.

   The key column uses fit-content so it hugs short keys but never grows past a
   cap; long, spaceless keys (file paths, identifiers) break inside it instead
   of overflowing under the divider. */
.kv {
  display: grid;
  grid-template-columns: fit-content(220px) 1fr;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.kv dt,
.kv dd {
  padding: var(--space-2) var(--space-4);
  border-top: 1px solid var(--border);
  overflow-wrap: anywhere;
}
.kv dt:first-of-type,
.kv dd:first-of-type {
  border-top: none;
}
.kv dt {
  background: var(--bg-surface);
  border-right: 1px solid var(--border);
}
.kv dd {
  margin: 0;
  background: var(--bg-elevated);
}
</style>
