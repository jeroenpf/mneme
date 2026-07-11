<script setup lang="ts">
import { computed } from 'vue'
import type { DocPhase } from '@/lib/phases'

const props = defineProps<{ phases: DocPhase[] }>()

const doneCount = computed(() => props.phases.filter((p) => p.status === 'done').length)
</script>

<template>
  <nav class="tracker">
    <div class="flex items-baseline gap-2">
      <h3 class="mn-label">phases</h3>
      <span class="mn-mono-sm">{{ doneCount }}/{{ phases.length }}</span>
    </div>
    <ol class="list">
      <li
        v-for="(phase, i) in phases"
        :key="i"
        class="row"
        :class="{ wip: phase.status === 'wip' }"
        data-test="phase-row"
      >
        <span class="pip" :class="`pip-${phase.status}`" />
        <span class="mn-mono-sm title">{{ phase.title }}</span>
      </li>
    </ol>
  </nav>
</template>

<style scoped>
.tracker {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  list-style: none;
  margin: 0;
  padding: 0;
}
.row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.pip {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: var(--radius-xs);
  background: var(--bg-overlay);
}
.pip-done { background: var(--accent); }
.pip-wip  { background: var(--accent-dim); box-shadow: inset 0 0 0 1px var(--accent-border); }

.title { color: var(--text-muted); }
.row.wip .title { color: var(--accent); }
</style>
