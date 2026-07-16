<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DocCard from '@/components/DocCard.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import StatsRow from '@/components/StatsRow.vue'
import { useDebounced } from '@/composables/useDebounced'
import { useDocuments } from '@/composables/useDocuments'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import { aggregateCounts, useProjects } from '@/composables/useProjects'
import {
  DEFAULT_STATUSES,
  PILL_STATUSES,
  sortDocuments,
  useRegistryFilters,
} from '@/composables/useRegistryFilters'

const { state, apiFilter, update } = useRegistryFilters()
const { items, loading, error, refresh } = useDocuments(apiFilter)
const { items: projects, refresh: refreshProjects } = useProjects()

const counts = computed(() => aggregateCounts(projects.value))

// Live updates: when the agent pushes/archives a document over MCP, silently
// refetch both the list and the project stats (grid stays mounted) and flash
// the changed/added card.
useLiveRefresh('documents', {
  refresh: () => Promise.all([refresh({ silent: true }), refreshProjects({ silent: true })]),
  flashTarget: (ev) => `[data-flash-id="${ev.id}"]`,
})

// Live search: local input, debounced 300ms into the URL (and thus the API).
const search = ref(state.value.q ?? '')
const debouncedSearch = useDebounced(search, 300)
watch(debouncedSearch, (q) => {
  if ((q || undefined) !== state.value.q) update({ q: q || undefined })
})
// Keep the input in sync when the URL changes underneath us (back/forward).
watch(
  () => state.value.q,
  (q) => {
    if ((q ?? '') !== search.value) search.value = q ?? ''
  },
)

// One fetch feeds both grids: unfiltered lists include archived docs,
// so the page partitions client-side. With a status filter active the
// archived list is empty and the section hides itself.
// Status is filtered here (client-side): the fetch returns the whole page
// including archived and complete; the active grid keeps only non-archived
// docs whose status is in the selected set. Archived stays in its own
// section, independent of the status pills.
const sorted = computed(() => sortDocuments(items.value, state.value.sort))
const statusSet = computed(() => new Set(state.value.statuses))
const active = computed(() =>
  sorted.value.filter((d) => d.status !== 'archived' && statusSet.value.has(d.status)),
)
const archived = computed(() => sorted.value.filter((d) => d.status === 'archived'))
const showArchived = ref(false)

// Selection differs from the default working set (or a project/type/q filter
// is active) — decides the empty-state copy (clear-filters vs. nothing-yet).
const isDefaultStatuses = computed(() => {
  const s = state.value.statuses
  return s.length === DEFAULT_STATUSES.length && DEFAULT_STATUSES.every((x) => s.includes(x))
})
const hasFilters = computed(
  () => Boolean(state.value.type || state.value.project || state.value.q) || !isDefaultStatuses.value,
)

function clearFilters() {
  search.value = ''
  update({ statuses: [...DEFAULT_STATUSES], type: undefined, project: undefined, q: undefined })
}

function showAll() {
  update({ statuses: [...PILL_STATUSES] })
}
</script>

<template>
  <div>
    <main class="mx-auto flex max-w-[1200px] flex-col gap-4 px-6 py-6">
      <StatsRow :counts="counts" />

      <div class="searchbar">
        <span class="mag" aria-hidden="true">⌕</span>
        <input
          v-model="search"
          type="search"
          class="search-input mn-body-sm"
          placeholder="Filter documents…"
          aria-label="Filter documents"
        />
      </div>

      <FilterToolbar :state="state" :projects="projects" @change="update" />

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading registry…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load documents: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      </div>

      <template v-else>
        <p
          v-if="state.statuses.length === 0"
          class="mn-body-sm py-8 text-center"
          data-test="no-statuses"
        >
          no statuses selected —
          <button class="link" @click="showAll">show all</button>
        </p>

        <p
          v-else-if="active.length === 0 && archived.length === 0"
          class="mn-body-sm py-8 text-center"
          data-test="empty"
        >
          <template v-if="hasFilters">
            no documents match —
            <button class="link" @click="clearFilters">clear filters</button>
          </template>
          <template v-else>nothing here yet — push a document over MCP</template>
        </p>

        <div v-else-if="active.length" class="doc-grid" data-test="grid">
          <DocCard v-for="d in active" :key="d.id" :doc="d" :data-flash-id="d.id" />
        </div>

        <section v-if="archived.length">
          <button
            class="archived-toggle mn-label"
            data-test="archived-toggle"
            @click="showArchived = !showArchived"
          >
            {{ showArchived ? '▾' : '▸' }} archived ({{ archived.length }})
          </button>
          <div v-if="showArchived" class="doc-grid archived-grid mt-3" data-test="archived-grid">
            <DocCard v-for="d in archived" :key="d.id" :doc="d" :data-flash-id="d.id" />
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
/* In-content live filter of the registry list (the rail owns global search).
   Wrapper carries the border + focus ring; the input itself is chromeless. */
.searchbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 9px var(--space-3);
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
}
.searchbar:focus-within {
  border-color: transparent;
  box-shadow: var(--shadow-focus);
}
.mag {
  color: var(--text-faint);
  font-size: 14px;
  line-height: 1;
}
.search-input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  color: var(--text-primary);
  padding: 0;
}
.search-input::placeholder {
  color: var(--text-faint);
}
.search-input:focus {
  outline: none;
}

.doc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-3);
}

.archived-grid :deep(.doc-card) {
  opacity: 0.55;
}
.archived-grid :deep(.doc-card:hover) {
  opacity: 0.85;
}

.archived-toggle {
  background: none;
  border: none;
  cursor: pointer;
  padding: var(--space-2) 0;
}
.archived-toggle:hover {
  color: var(--text-secondary);
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

.link {
  background: none;
  border: none;
  padding: 0;
  color: var(--accent);
  cursor: pointer;
  font: inherit;
}
.link:hover {
  color: var(--accent-hover);
}
</style>
