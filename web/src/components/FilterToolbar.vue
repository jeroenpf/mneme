<script setup lang="ts">
import type { DocumentStatus, DocumentType, ProjectStats } from '@/types'
import type { RegistryFilterState, SortKey } from '@/composables/useRegistryFilters'

const props = defineProps<{ state: RegistryFilterState; projects: ProjectStats[] }>()
const emit = defineEmits<{ change: [patch: Partial<RegistryFilterState>] }>()

const STATUS_PILLS: DocumentStatus[] = ['todo', 'in-progress', 'complete', 'blocked']
const TYPES: DocumentType[] = ['plan', 'report', 'spec', 'adr', 'brainstorm', 'journal']
const SORTS: SortKey[] = ['updated', 'created', 'title']

function toggleStatus(s: DocumentStatus) {
  emit('change', { status: props.state.status === s ? undefined : s })
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
    <button
      class="pill"
      :class="{ active: !state.status }"
      @click="emit('change', { status: undefined })"
    >
      all
    </button>
    <button
      v-for="s in STATUS_PILLS"
      :key="s"
      class="pill"
      :class="{ active: state.status === s }"
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

.pill:focus-visible,
.select:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
</style>
