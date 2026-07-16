<script setup lang="ts">
import { computed, ref } from 'vue'
import { groupJournal, useJournal } from '@/composables/useJournal'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import JournalEntryCard from '@/components/JournalEntryCard.vue'

const { items, loading, error, refresh } = useJournal()

// Live updates: when the agent appends a journal entry over MCP, silently
// refetch (list stays mounted) and flash the new entry.
useLiveRefresh('journal', {
  refresh: () => refresh({ silent: true }),
  flashTarget: (ev) => `[data-flash-id="${ev.id}"]`,
})

const projectFilter = ref('') // '' = all

const filtered = computed(() =>
  items.value.filter((e) => projectFilter.value === '' || (e.project ?? '') === projectFilter.value),
)
const groups = computed(() => groupJournal(filtered.value))
const isEmpty = computed(() => items.value.length === 0)

const projects = computed(() => {
  const set = new Set<string>()
  for (const e of items.value) set.add(e.project ?? '')
  return [...set].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b)))
})

const projectLabel = (slug: string) => (slug === '' ? 'global' : slug)
</script>

<template>
  <div>
    <main class="content">
      <p class="mn-body-sm intro">
        The dev journal — what each session built, deferred, and changed. Claude Code writes it with
        <span class="mn-code-inline">append_journal</span> and reads it with
        <span class="mn-code-inline">get_journal</span>. Read-only here.
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading journal…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load the journal: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      </div>

      <template v-else>
        <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
          no journal entries yet — write one over MCP with
          <span class="mn-code-inline">append_journal</span>.
        </p>

        <template v-else>
          <div class="filters">
            <select v-model="projectFilter" class="filter mn-mono-sm" aria-label="filter by project" data-test="project-filter">
              <option value="">all projects</option>
              <option v-for="p in projects" :key="p || '_global'" :value="p">{{ projectLabel(p) }}</option>
            </select>
          </div>

          <section v-for="group in groups" :key="group.project || '_global'" class="group" data-test="group">
            <h2 class="group-title mn-h3">
              <span class="scope-prefix mn-mono-sm">project /</span> {{ projectLabel(group.project) }}
            </h2>
            <div class="cards">
              <JournalEntryCard
                v-for="e in group.entries"
                :key="e.id"
                :entry="e"
                :data-flash-id="e.id"
              />
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
  flex-wrap: wrap;
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
.cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
