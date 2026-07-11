<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DocCard from '@/components/DocCard.vue'
import FilterToolbar from '@/components/FilterToolbar.vue'
import StatsRow from '@/components/StatsRow.vue'
import Topbar from '@/components/Topbar.vue'
import { useDebounced } from '@/composables/useDebounced'
import { useDocuments } from '@/composables/useDocuments'
import { aggregateCounts, useProjects } from '@/composables/useProjects'
import { sortDocuments, useRegistryFilters } from '@/composables/useRegistryFilters'

const { state, apiFilter, update } = useRegistryFilters()
const { items, loading, error, refresh } = useDocuments(apiFilter)
const { items: projects } = useProjects()

const counts = computed(() => aggregateCounts(projects.value))

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
const sorted = computed(() => sortDocuments(items.value, state.value.sort))
const active = computed(() => sorted.value.filter((d) => d.status !== 'archived'))
const archived = computed(() => sorted.value.filter((d) => d.status === 'archived'))
const showArchived = ref(false)

const hasFilters = computed(() =>
  Boolean(state.value.status || state.value.type || state.value.project || state.value.q),
)

function clearFilters() {
  search.value = ''
  update({ status: undefined, type: undefined, project: undefined, q: undefined })
}
</script>

<template>
  <div>
    <Topbar v-model="search" />

    <main class="mx-auto flex max-w-[1200px] flex-col gap-4 px-6 py-6">
      <StatsRow :counts="counts" />
      <FilterToolbar :state="state" :projects="projects" @change="update" />

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading registry…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load documents: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh">retry</button>
      </div>

      <template v-else>
        <p
          v-if="active.length === 0 && archived.length === 0"
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
          <DocCard v-for="d in active" :key="d.id" :doc="d" />
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
            <DocCard v-for="d in archived" :key="d.id" :doc="d" />
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
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
