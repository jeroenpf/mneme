<script setup lang="ts">
import { computed } from 'vue'
import type { DocPhase } from '@/lib/phases'

const props = defineProps<{ phases: DocPhase[] }>()

const doneCount = computed(() => props.phases.filter((p) => p.status === 'done').length)

// Spine state class — the mockup names the active phase "current", not "wip".
function stateClass(status: DocPhase['status']): 'done' | 'current' | 'todo' {
  return status === 'wip' ? 'current' : status
}

const TAG: Record<DocPhase['status'], string> = {
  done: 'done',
  wip: 'in progress',
  todo: 'todo',
}
</script>

<template>
  <nav class="tracker">
    <div class="head">
      <h3 class="mn-label">phases</h3>
      <span class="mn-mono-sm count">{{ doneCount }}/{{ phases.length }}</span>
    </div>
    <ol class="spine">
      <li
        v-for="(phase, i) in phases"
        :key="i"
        class="phase"
        :class="stateClass(phase.status)"
        data-test="phase-row"
      >
        <span class="node">
          <span v-if="phase.status === 'done'" class="check" aria-hidden="true">✓</span>
        </span>
        <span class="p-name">
          {{ phase.title }}
          <span class="p-tag">{{ TAG[phase.status] }}</span>
        </span>
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
.head {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}
.count {
  color: var(--text-faint);
}

/* Connected phase spine: a vertical rail threads the nodes; each node punches
   through it (node bg = page bg + a bg-colored ring) so the line reads as a
   continuous connector between states. */
.spine {
  position: relative;
  display: flex;
  flex-direction: column;
  list-style: none;
  margin: 0;
  padding: 0;
}
.phase {
  position: relative;
  display: grid;
  grid-template-columns: 22px 1fr;
  gap: var(--space-3);
  padding-bottom: var(--space-4);
}
.phase:last-child {
  padding-bottom: 0;
}
/* the connector — behind the node, hidden on the last row */
.phase::before {
  content: '';
  position: absolute;
  left: 10px;
  top: 20px;
  bottom: -2px;
  width: 2px;
  background: var(--border-strong);
}
.phase:last-child::before {
  display: none;
}
.phase.done::before {
  background: var(--green);
  opacity: 0.55;
}

.node {
  width: 21px;
  height: 21px;
  border-radius: var(--radius-pill);
  display: grid;
  place-items: center;
  z-index: 1;
  background: var(--bg);
}
.check {
  font-size: 11px;
  line-height: 1;
  /* light in the light themes, dark in Ink — always the inverse of the
     green node, so no hardcoded on-color needed */
  color: var(--bg-elevated);
}
.phase.done .node {
  background: var(--green);
  box-shadow: 0 0 0 3px var(--bg);
}
.phase.current .node {
  background: var(--accent);
  box-shadow:
    0 0 0 3px var(--bg),
    0 0 0 5px var(--accent-border);
}
.phase.current .node::after {
  content: '';
  width: 7px;
  height: 7px;
  border-radius: var(--radius-pill);
  background: var(--bg);
}
.phase.todo .node {
  background: var(--bg);
  box-shadow: inset 0 0 0 2px var(--border-strong);
}

.p-name {
  padding-top: 2px;
  font-family: var(--font-body);
  font-size: var(--fs-body-sm);
  line-height: 1.35;
  color: var(--text-secondary);
}
.phase.current .p-name {
  color: var(--text-primary);
  font-weight: 600;
}
.phase.done .p-name {
  color: var(--text-muted);
}
.p-tag {
  display: block;
  margin-top: 3px;
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: var(--tracking-label);
  text-transform: uppercase;
  color: var(--text-faint);
}
.phase.current .p-tag {
  color: var(--accent);
}
</style>
