<script setup lang="ts">
import { computed } from 'vue'
import type { DocumentStatus, DocumentType, ProjectStats } from '@/types'
import { PILL_STATUSES, type RegistryFilterState, type SortKey } from '@/composables/useRegistryFilters'
import type { ViewMode } from '@/composables/useViewMode'

const props = defineProps<{ state: RegistryFilterState; projects: ProjectStats[]; view: ViewMode }>()
const emit = defineEmits<{
  change: [patch: Partial<RegistryFilterState>]
  viewChange: [v: ViewMode]
}>()

const STATUS_PILLS: readonly DocumentStatus[] = PILL_STATUSES
const TYPES: DocumentType[] = ['plan', 'report', 'spec', 'adr', 'brainstorm', 'journal']
const SORTS: SortKey[] = ['updated', 'created', 'title']

const allActive = computed(() => STATUS_PILLS.every((s) => props.state.statuses.includes(s)))

function isActive(s: DocumentStatus): boolean {
  return props.state.statuses.includes(s)
}

// Toggle membership, re-emitting the whole selection in canonical pill order.
function toggleStatus(s: DocumentStatus) {
  const next = new Set(props.state.statuses)
  next.has(s) ? next.delete(s) : next.add(s)
  emit('change', { statuses: STATUS_PILLS.filter((x) => next.has(x)) })
}

function selectAll() {
  emit('change', { statuses: [...STATUS_PILLS] })
}

function onProject(e: Event) {
  const v = (e.target as HTMLSelectElement).value
  emit('change', { project: v || undefined })
}

function onType(e: Event) {
  const v = (e.target as HTMLSelectElement).value
  emit('change', { type: (v || undefined) as DocumentType | undefined })
}

function onSort(e: Event) {
  emit('change', { sort: (e.target as HTMLSelectElement).value as SortKey })
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-2">
    <button class="pill" :class="{ active: allActive }" @click="selectAll">all</button>
    <button
      v-for="s in STATUS_PILLS"
      :key="s"
      class="pill"
      :class="{ active: isActive(s) }"
      @click="toggleStatus(s)"
    >
      {{ s }}
    </button>

    <div class="ml-auto flex items-center gap-2">
      <select
        class="select"
        aria-label="Filter by project"
        :value="state.project ?? ''"
        @change="onProject"
      >
        <option value="">all projects</option>
        <option v-for="p in projects" :key="p.slug" :value="p.slug">{{ p.slug }}</option>
      </select>

      <select class="select" aria-label="Filter by type" :value="state.type ?? ''" @change="onType">
        <option value="">all types</option>
        <option v-for="t in TYPES" :key="t" :value="t">{{ t }}</option>
      </select>

      <select class="select" aria-label="Sort by" :value="state.sort" @change="onSort">
        <option v-for="s in SORTS" :key="s" :value="s">sort: {{ s }}</option>
      </select>

      <div class="viewtoggle" role="group" aria-label="View">
        <button
          class="viewbtn"
          :class="{ active: view === 'cards' }"
          aria-label="Card view"
          data-test="view-cards"
          @click="emit('viewChange', 'cards')"
        >
          ⊞
        </button>
        <button
          class="viewbtn"
          :class="{ active: view === 'list' }"
          aria-label="List view"
          data-test="view-list"
          @click="emit('viewChange', 'list')"
        >
          ≡
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pill {
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  line-height: var(--lh-mono-sm);
  color: var(--text-muted);
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  cursor: pointer;
  transition:
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}
.pill:hover {
  background: var(--bg-hover);
  color: var(--text-secondary);
}
.pill.active {
  color: var(--accent);
  background: var(--accent-dim);
  border-color: var(--accent-border);
}

.select {
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  color: var(--text-secondary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 4px 8px;
}

.viewtoggle {
  display: flex;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.viewbtn {
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  line-height: var(--lh-mono-sm);
  color: var(--text-muted);
  background: transparent;
  border: none;
  padding: 4px 8px;
  cursor: pointer;
  transition:
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}
.viewbtn + .viewbtn {
  border-left: 1px solid var(--border);
}
.viewbtn:hover {
  background: var(--bg-hover);
  color: var(--text-secondary);
}
.viewbtn.active {
  color: var(--accent);
  background: var(--accent-dim);
}

.pill:focus-visible,
.select:focus-visible,
.viewbtn:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
</style>
