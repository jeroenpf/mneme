<script setup lang="ts">
import { computed } from 'vue'
import type { RegistryCounts } from '@/composables/useProjects'

const props = defineProps<{ counts: RegistryCounts }>()

const cells = computed(() => [
  { label: 'total', value: props.counts.total, tone: 'primary' },
  { label: 'in progress', value: props.counts.inProgress, tone: 'wip' },
  { label: 'complete', value: props.counts.complete, tone: 'done' },
  { label: 'todo', value: props.counts.todo, tone: 'todo' },
])
</script>

<template>
  <div class="stats-strip">
    <div v-for="cell in cells" :key="cell.label" class="cell" data-test="stat-cell">
      <div class="mn-label">{{ cell.label }}</div>
      <div class="num" :class="`tone-${cell.tone}`">{{ cell.value }}</div>
    </div>
  </div>
</template>

<style scoped>
.stats-strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
}
.cell + .cell {
  border-left: 1px solid var(--border-soft);
}
.num {
  font-family: var(--font-mono);
  font-size: var(--fs-h1);
  line-height: var(--lh-h1);
  font-weight: 600;
}
.tone-primary { color: var(--text-primary); }
.tone-wip     { color: var(--status-wip); }
.tone-done    { color: var(--status-done); }
.tone-todo    { color: var(--text-muted); }
</style>
