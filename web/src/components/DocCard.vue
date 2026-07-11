<script setup lang="ts">
import { computed } from 'vue'
import type { Document } from '@/types'

const props = defineProps<{ doc: Document }>()

const description = computed(() => {
  const d = props.doc.meta?.description
  return typeof d === 'string' ? d : ''
})

// phase_current is the phase in flight (1-based): phases below it are
// done, it is the active segment, the rest are todo.
const pips = computed(() => {
  const total = props.doc.phase_total ?? 0
  const current = props.doc.phase_current ?? 0
  return Array.from({ length: total }, (_, i) =>
    i + 1 < current ? 'done' : i + 1 === current ? 'wip' : 'todo',
  )
})
</script>

<template>
  <RouterLink :to="`/doc/${doc.id}`" class="doc-card">
    <div class="flex items-center gap-2">
      <span class="status-dot" :class="`status-${doc.status}`" :title="doc.status" />
      <span class="mn-label">{{ doc.type }}</span>
      <span v-if="doc.ticket" class="mn-mono-sm ml-auto">{{ doc.ticket }}</span>
    </div>

    <h3 class="mn-h2 line-clamp-2">{{ doc.title }}</h3>

    <p v-if="description" class="mn-body-sm line-clamp-2">{{ description }}</p>

    <div v-if="pips.length" class="flex items-center gap-1" data-test="pips">
      <span v-for="(p, i) in pips" :key="i" class="pip" :class="`pip-${p}`" />
      <span class="mn-mono-sm ml-1">{{ doc.phase_current }}/{{ doc.phase_total }}</span>
    </div>

    <div v-if="doc.tags.length || doc.repo" class="mt-auto flex items-baseline gap-2">
      <span v-for="tag in doc.tags" :key="tag" class="mn-mono-sm">#{{ tag }}</span>
      <span v-if="doc.repo" class="mn-mono-sm ml-auto text-text-faint" data-test="repo">
        ⎇ {{ doc.repo }}
      </span>
    </div>
  </RouterLink>
</template>

<style scoped>
.doc-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}
.doc-card:hover {
  background: var(--bg-elevated);
  border-color: var(--border-strong);
}
.doc-card:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}

.status-dot {
  width: 7px;
  height: 7px;
  flex: none;
  border-radius: var(--radius-xs);
}
.status-todo        { background: var(--status-todo); }
.status-in-progress { background: var(--status-wip); }
.status-complete    { background: var(--status-done); }
.status-blocked     { background: var(--status-blocked); }
.status-archived    { background: var(--status-archived); }

.pip {
  width: 14px;
  height: 4px;
  border-radius: 1px;
  background: var(--bg-overlay);
}
.pip-done { background: var(--accent); }
.pip-wip  { background: var(--accent-dim); box-shadow: inset 0 0 0 1px var(--accent-border); }
</style>
