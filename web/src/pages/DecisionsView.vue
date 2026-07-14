<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Decision, DecisionStatus } from '@/api/decisions'
import { groupDecisions, useDecisions } from '@/composables/useDecisions'

const { items, loading, error, refresh } = useDecisions()

const projectFilter = ref('') // '' = all
const statusFilter = ref<'' | DecisionStatus>('')

const filtered = computed(() =>
  items.value.filter(
    (d) =>
      (projectFilter.value === '' || (d.project ?? '') === projectFilter.value) &&
      (statusFilter.value === '' || d.status === statusFilter.value),
  ),
)
const groups = computed(() => groupDecisions(filtered.value))
const isEmpty = computed(() => items.value.length === 0)

// Distinct project buckets for the filter dropdown ('' = global).
const projects = computed(() => {
  const set = new Set<string>()
  for (const d of items.value) set.add(d.project ?? '')
  return [...set].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b)))
})

const open = ref(new Set<string>())
function toggle(id: string) {
  const next = new Set(open.value)
  next.has(id) ? next.delete(id) : next.add(id)
  open.value = next
}

const statusDot: Record<DecisionStatus, string> = {
  proposed: 'status-wip',
  accepted: 'status-done',
  deprecated: 'status-archived',
}
const projectLabel = (slug: string) => (slug === '' ? 'global' : slug)
const fmtDate = (iso: string) => iso.slice(0, 10)
const hasDetail = (d: Decision) =>
  Boolean(d.rationale || d.alternatives || d.consequences)
</script>

<template>
  <div>
    <main class="content">
      <p class="mn-body-sm intro">
        The decision log — why things are the way they are. Claude Code writes it with
        <span class="mn-code-inline">log_decision</span> and searches it with
        <span class="mn-code-inline">query_decisions</span>. Read-only here.
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading decisions…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load decisions: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh">retry</button>
      </div>

      <template v-else>
        <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
          no decisions yet — record one over MCP with
          <span class="mn-code-inline">log_decision</span>.
        </p>

        <template v-else>
          <div class="filters">
            <select v-model="projectFilter" class="filter mn-mono-sm" aria-label="filter by project" data-test="project-filter">
              <option value="">all projects</option>
              <option v-for="p in projects" :key="p || '_global'" :value="p">{{ projectLabel(p) }}</option>
            </select>
            <select v-model="statusFilter" class="filter mn-mono-sm" aria-label="filter by status" data-test="status-filter">
              <option value="">all statuses</option>
              <option value="proposed">proposed</option>
              <option value="accepted">accepted</option>
              <option value="deprecated">deprecated</option>
            </select>
          </div>

          <section v-for="group in groups" :key="group.project || '_global'" class="group" data-test="group">
            <h2 class="group-title mn-h3">
              <span class="scope-prefix mn-mono-sm">project /</span> {{ projectLabel(group.project) }}
            </h2>

            <div class="timeline">
              <article
                v-for="d in group.decisions"
                :key="d.id"
                class="decision"
                :data-test="`decision-${d.id}`"
              >
                <button
                  class="decision-head"
                  type="button"
                  data-test="decision-toggle"
                  :aria-expanded="open.has(d.id)"
                  @click="toggle(d.id)"
                >
                  <span class="status-dot" :class="statusDot[d.status]" :title="d.status" />
                  <span class="decision-title mn-body">{{ d.title }}</span>
                  <span class="decision-status mn-mono-sm">{{ d.status }}</span>
                  <time class="decision-date mn-mono-sm">{{ fmtDate(d.created_at) }}</time>
                </button>

                <div v-if="open.has(d.id)" class="detail" :data-test="`detail-${d.id}`">
                  <p class="detail-decision mn-body-sm">{{ d.decision }}</p>
                  <template v-if="d.rationale">
                    <h3 class="detail-label mn-label">rationale</h3>
                    <p class="mn-body-sm">{{ d.rationale }}</p>
                  </template>
                  <template v-if="d.alternatives">
                    <h3 class="detail-label mn-label">alternatives</h3>
                    <p class="mn-body-sm">{{ d.alternatives }}</p>
                  </template>
                  <template v-if="d.consequences">
                    <h3 class="detail-label mn-label">consequences</h3>
                    <p class="mn-body-sm">{{ d.consequences }}</p>
                  </template>
                  <p v-if="!hasDetail(d)" class="mn-mono-sm detail-empty">— no rationale recorded —</p>
                </div>
              </article>
            </div>
          </section>
        </template>
      </template>
    </main>
  </div>
</template>

<style scoped>
.content {
  max-width: var(--content-max);
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  min-width: 0;
}
.intro {
  color: var(--text-muted);
  margin-top: calc(-1 * var(--space-2));
}
.error {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.retry {
  margin-top: var(--space-2);
  padding: 4px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}
.empty {
  color: var(--text-muted);
}

.filters {
  display: flex;
  gap: var(--space-3);
}
.filter {
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.filter:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
}

.group {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.group-title {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}
.scope-prefix {
  color: var(--text-faint);
}

.timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  border-left: 1px solid var(--border-soft);
  padding-left: var(--space-4);
}
.decision {
  display: flex;
  flex-direction: column;
}
.decision-head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2) 0;
  background: transparent;
  border: none;
  text-align: left;
  cursor: pointer;
  color: var(--text-primary);
}
.status-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
}
.status-wip {
  background: var(--status-wip);
}
.status-done {
  background: var(--status-done);
}
.status-archived {
  background: var(--status-archived);
}
.decision-title {
  overflow-wrap: anywhere;
}
.decision-status {
  color: var(--text-faint);
}
.decision-date {
  color: var(--text-faint);
}
.detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: 0 0 var(--space-3) var(--space-5);
  color: var(--text-secondary);
}
.detail-decision {
  color: var(--text-primary);
}
.detail-label {
  margin-top: var(--space-2);
  color: var(--text-faint);
}
.detail-empty {
  color: var(--text-faint);
}
</style>
