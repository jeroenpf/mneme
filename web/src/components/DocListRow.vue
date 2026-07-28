<script setup lang="ts">
import { computed } from 'vue'
import PhasePips from '@/components/PhasePips.vue'
import { timeAgo } from '@/lib/time'
import type { Document } from '@/types'

const props = defineProps<{ doc: Document; now: number }>()

const ago = computed(() => timeAgo(props.doc.updated_at, props.now))
const exact = computed(() => new Date(props.doc.updated_at).toLocaleString())
</script>

<template>
  <RouterLink :to="`/doc/${doc.id}`" class="doc-row">
    <span class="badge mn-mono-sm" :class="`status-${doc.status}`" data-test="badge">
      {{ doc.status }}
    </span>
    <span class="mn-label">{{ doc.type }}</span>
    <span class="title mn-body-sm">{{ doc.title }}</span>
    <span class="cell">
      <PhasePips v-if="doc.phase_total" :current="doc.phase_current ?? 0" :total="doc.phase_total" />
    </span>
    <span class="cell mn-mono-sm">{{ doc.ticket }}</span>
    <span class="cell mn-mono-sm">{{ doc.project }}</span>
    <time class="cell mn-mono-sm updated" data-test="updated" :title="exact">{{ ago }}</time>
  </RouterLink>
</template>

<style scoped>
/* One row = one subgrid item spanning the container's 7 tracks; identical
   padding on every row keeps the shared tracks aligned. */
.doc-row {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: subgrid;
  align-items: center;
  padding: var(--space-2) var(--space-3);
  text-decoration: none;
  transition: background var(--duration-fast) var(--ease-out);
}
.doc-row:hover {
  background: var(--bg-elevated);
}
.doc-row:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.doc-row + .doc-row {
  border-top: 1px solid var(--border);
}

/* Text tinted by the --status-* token; bg/border derived from it via
   color-mix so the badge works across all themes with zero new tokens. */
.badge {
  justify-self: start;
  padding: 1px 8px;
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, currentColor 12%, transparent);
  border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
}
.status-todo        { color: var(--status-todo); }
.status-in-progress { color: var(--status-wip); }
.status-complete    { color: var(--status-done); }
.status-blocked     { color: var(--status-blocked); }
.status-archived    { color: var(--status-archived); }

.title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
}
.cell {
  color: var(--text-muted);
}
.updated {
  color: var(--text-faint);
}
</style>
