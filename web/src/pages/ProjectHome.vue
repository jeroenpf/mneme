<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { getBundle, type Bundle } from '@/api/bundle'
import { listProjects } from '@/api/projects'
import RefChip from '@/components/RefChip.vue'
import StatsRow from '@/components/StatsRow.vue'
import type { RegistryCounts } from '@/composables/useProjects'
import type { ProjectStats } from '@/types'

// The project home answers "where does work stand?" from a single context
// bundle: the active plan, what to do next, what is blocked, what was decided,
// and what the last session handed off. Mutations still happen over MCP — this
// is a read-mostly workflow landing that deep-links into the viewer.
const props = defineProps<{ slug: string }>()
const router = useRouter()

const bundle = ref<Bundle | null>(null)
const loading = ref(true)
const error = ref<Error | null>(null)
const projects = ref<ProjectStats[]>([])

async function load(slug: string) {
  loading.value = true
  error.value = null
  try {
    bundle.value = await getBundle(slug)
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
    bundle.value = null
  } finally {
    loading.value = false
  }
}

watch(() => props.slug, load, { immediate: true })
listProjects()
  .then((p) => (projects.value = p))
  .catch(() => {})

const plan = computed(() => bundle.value?.active_plan ?? null)
const nextTasks = computed(() => bundle.value?.next_tasks ?? [])
const blockers = computed(() => bundle.value?.blockers ?? [])
const decisions = computed(() => bundle.value?.decisions ?? [])
// The handoff is the most recent session: its summary plus the work it parked.
const handoff = computed(() => bundle.value?.journal?.[0] ?? null)

// Plan lifecycle counts feed the reused StatsRow strip.
const counts = computed<RegistryCounts>(() => {
  const s = bundle.value?.plan_stats
  return {
    total: s?.total ?? 0,
    inProgress: s?.in_progress ?? 0,
    complete: s?.complete ?? 0,
    todo: s?.todo ?? 0,
  }
})

// Fallback line when nothing is in flight — tells the reader whether work is
// finished, waiting to start, or not begun (mirrors the bundle's own fallback).
const noPlanMessage = computed(() => {
  const s = bundle.value?.plan_stats
  if (!s || s.total === 0) return 'No plans yet — push one over MCP to start tracking work.'
  if (s.complete === s.total) return `All ${s.total} plans complete.`
  if (s.todo > 0) return `No plan in progress — ${s.todo} waiting to start.`
  return 'No plan in progress.'
})

function switchProject(e: Event) {
  const slug = (e.target as HTMLSelectElement).value
  if (slug && slug !== props.slug) router.push(`/project/${slug}`)
}
</script>

<template>
  <main class="home">
    <header class="head">
      <div class="titles">
        <p class="mn-label eyebrow">project home</p>
        <h1 class="mn-h1" data-test="project-name">{{ slug }}</h1>
      </div>
      <select
        v-if="projects.length > 1"
        class="picker mn-mono-sm"
        aria-label="switch project"
        data-test="project-picker"
        :value="slug"
        @change="switchProject"
      >
        <option v-for="p in projects" :key="p.slug" :value="p.slug">{{ p.name }}</option>
      </select>
    </header>

    <p v-if="loading" class="mn-mono-sm state" data-test="loading">loading…</p>

    <div v-else-if="error" class="error-box mn-body-sm" data-test="error">
      could not load this project: {{ error.message }}
      <button class="retry mn-mono-sm" @click="load(slug)">retry</button>
    </div>

    <template v-else-if="bundle">
      <!-- Current work: the plan in flight and its active phase. -->
      <RouterLink
        v-if="plan"
        class="current-work"
        data-test="current-work"
        :to="`/doc/${plan.id}`"
      >
        <div class="cw-top">
          <span class="mn-label">current work</span>
          <span class="badge" :class="`status-${plan.status}`">{{ plan.status }}</span>
        </div>
        <h2 class="mn-h2">{{ plan.title }}</h2>
        <p v-if="plan.phase_total" class="cw-phase mn-body-sm">
          <span v-if="plan.active_phase" class="phase-name">{{ plan.active_phase }}</span>
          <span class="mn-mono-sm phase-count">phase {{ plan.phase_current }}/{{ plan.phase_total }}</span>
        </p>
      </RouterLink>
      <p v-else class="empty-plan mn-body-sm" data-test="no-active-plan">{{ noPlanMessage }}</p>

      <StatsRow :counts="counts" />

      <div class="panels">
        <!-- Next tasks: the leading incomplete tasks of the active plan. -->
        <section class="panel" data-test="next-tasks">
          <h3 class="mn-label panel-head">next tasks</h3>
          <ul v-if="nextTasks.length && plan" class="rows">
            <li v-for="t in nextTasks" :key="t.id" class="row" data-test="next-task">
              <RouterLink class="row-main" :to="{ path: `/doc/${plan.id}`, hash: `#${t.id}` }">
                <span class="row-title mn-body-sm">{{ t.title }}</span>
                <span v-if="t.phase" class="row-sub mn-mono-sm">{{ t.phase }}</span>
              </RouterLink>
              <code class="tid mn-mono-sm">{{ t.id }}</code>
            </li>
          </ul>
          <p v-else class="panel-empty mn-body-sm">No open tasks in the active plan.</p>
        </section>

        <!-- Blockers: documents parked in the blocked state. -->
        <section class="panel" data-test="blockers">
          <h3 class="mn-label panel-head">blockers</h3>
          <ul v-if="blockers.length" class="rows">
            <li v-for="b in blockers" :key="b.id" class="row" data-test="blocker">
              <RouterLink class="row-main" :to="`/doc/${b.id}`">
                <span class="row-title mn-body-sm">{{ b.title }}</span>
              </RouterLink>
            </li>
          </ul>
          <p v-else class="panel-empty mn-body-sm">Nothing is blocked.</p>
        </section>

        <!-- Recent decisions: the ADR log's latest entries with rationale. -->
        <section class="panel" data-test="decisions">
          <h3 class="mn-label panel-head">recent decisions</h3>
          <ul v-if="decisions.length" class="rows">
            <li v-for="d in decisions" :key="d.id" class="row" data-test="decision">
              <RouterLink class="row-main" :to="{ path: '/decisions', query: { flash: d.public_id } }">
                <span class="row-title mn-body-sm">
                  {{ d.title }}
                  <span class="dec-status mn-mono-sm" :class="`dstatus-${d.status}`">{{ d.status }}</span>
                </span>
                <span v-if="d.rationale" class="row-sub mn-body-sm">{{ d.rationale }}</span>
              </RouterLink>
              <RefChip :public-id="d.public_id" kind="decision" compact />
            </li>
          </ul>
          <p v-else class="panel-empty mn-body-sm">No decisions logged yet.</p>
        </section>

        <!-- Handoff: what the last session did and consciously left for later. -->
        <section class="panel" data-test="handoff">
          <h3 class="mn-label panel-head">handoff</h3>
          <template v-if="handoff">
            <RouterLink
              class="handoff-main"
              :to="{ path: '/journal', query: { flash: handoff.public_id } }"
            >
              <span class="row-sub mn-mono-sm">{{ handoff.session_ref }}</span>
              <span class="row-title mn-body-sm">{{ handoff.summary }}</span>
            </RouterLink>
            <div v-if="handoff.deferred?.length" class="deferred">
              <p class="mn-label deferred-head">deferred</p>
              <ul class="rows">
                <li v-for="(d, i) in handoff.deferred" :key="i" class="deferred-item mn-body-sm">{{ d }}</li>
              </ul>
            </div>
          </template>
          <p v-else class="panel-empty mn-body-sm">No session handoff recorded.</p>
        </section>
      </div>
    </template>
  </main>
</template>

<style scoped>
.home {
  max-width: 1100px;
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-4);
}
.eyebrow {
  color: var(--eyebrow);
  margin-bottom: var(--space-1);
}
.picker {
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.picker:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.state {
  color: var(--text-muted);
  padding: var(--space-8) 0;
  text-align: center;
}
.error-box {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.retry {
  margin-left: var(--space-3);
  padding: 3px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}

/* Current-work banner — the one loud element: the plan in flight. */
.current-work {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-5);
  background: var(--bg-surface);
  border: 1px solid var(--accent-border);
  border-left: 3px solid var(--accent);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: inherit;
  transition: background var(--duration-fast) var(--ease-out);
}
.current-work:hover {
  background: var(--bg-elevated);
}
.current-work:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.cw-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.badge {
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  text-transform: uppercase;
  letter-spacing: var(--tracking-label);
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  color: var(--text-secondary);
  background: var(--bg-overlay);
}
.badge.status-in-progress {
  color: var(--status-wip);
  background: var(--accent-dim);
}
.badge.status-complete {
  color: var(--status-done);
  background: var(--green-dim);
}
.badge.status-blocked {
  color: var(--status-blocked);
  background: var(--red-dim);
}
.cw-phase {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  color: var(--text-secondary);
}
.phase-name {
  color: var(--accent);
  font-weight: 600;
}
.phase-count {
  color: var(--text-faint);
}
.empty-plan {
  padding: var(--space-4) var(--space-5);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-muted);
}

.panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
}
@media (max-width: 720px) {
  .panels {
    grid-template-columns: 1fr;
  }
}
.panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  min-width: 0;
}
.panel-head {
  color: var(--text-muted);
}
.panel-empty {
  color: var(--text-faint);
}
.rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.row {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  justify-content: space-between;
}
.row-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-decoration: none;
  color: inherit;
  min-width: 0;
  flex: 1;
}
.row-main:hover .row-title {
  color: var(--accent);
}
.row-title {
  color: var(--text-primary);
}
.row-sub {
  color: var(--text-muted);
}
.tid {
  color: var(--text-faint);
  flex: none;
  padding-top: 2px;
}
.dec-status {
  text-transform: uppercase;
  letter-spacing: var(--tracking-label);
  font-size: 9px;
  margin-left: var(--space-1);
  color: var(--text-faint);
}
.dstatus-accepted {
  color: var(--green);
}
.dstatus-deprecated {
  color: var(--text-faint);
  text-decoration: line-through;
}
.handoff-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-decoration: none;
  color: inherit;
}
.handoff-main:hover .row-title {
  color: var(--accent);
}
.deferred {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin-top: var(--space-1);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-soft);
}
.deferred-head {
  color: var(--text-faint);
}
.deferred-item {
  color: var(--text-secondary);
  padding-left: var(--space-3);
  position: relative;
}
.deferred-item::before {
  content: '→';
  position: absolute;
  left: 0;
  color: var(--text-faint);
}
</style>
