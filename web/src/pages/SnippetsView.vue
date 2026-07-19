<script setup lang="ts">
import { computed, ref } from 'vue'
import { groupSnippets, useSnippets } from '@/composables/useSnippets'
import { useDeepLinkFlash } from '@/composables/useDeepLinkFlash'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import SnippetCard from '@/components/SnippetCard.vue'

const { items, loading, error, refresh } = useSnippets()

// Search deep-links (?flash=<id>) scroll to and flash the matching row.
useDeepLinkFlash(() => !loading.value)

// Live updates: when the agent saves a snippet over MCP, silently refetch
// (list stays mounted) and flash the changed/added card.
useLiveRefresh('snippets', {
  refresh: () => refresh({ silent: true }),
  flashTarget: (ev) => `[data-flash-id="${ev.id}"]`,
})

const projectFilter = ref('') // '' = all
const languageFilter = ref('') // '' = all
const tagFilter = ref('') // '' = all

const filtered = computed(() =>
  items.value.filter(
    (s) =>
      (projectFilter.value === '' || (s.project ?? '') === projectFilter.value) &&
      (languageFilter.value === '' || s.language === languageFilter.value) &&
      (tagFilter.value === '' || s.tags.includes(tagFilter.value)),
  ),
)
const groups = computed(() => groupSnippets(filtered.value))
const isEmpty = computed(() => items.value.length === 0)

// Distinct filter option sets, derived from the loaded snippets.
const projects = computed(() => {
  const set = new Set<string>()
  for (const s of items.value) set.add(s.project ?? '')
  return [...set].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b)))
})
const languages = computed(() => {
  const set = new Set<string>()
  for (const s of items.value) if (s.language) set.add(s.language)
  return [...set].sort()
})
const tags = computed(() => {
  const set = new Set<string>()
  for (const s of items.value) for (const t of s.tags) set.add(t)
  return [...set].sort()
})

const projectLabel = (slug: string) => (slug === '' ? 'global' : slug)
</script>

<template>
  <div>
    <main class="content">
      <p class="mn-body-sm intro">
        The snippet library — reusable code patterns and project conventions. Claude Code writes it with
        <span class="mn-code-inline">save_snippet</span> and finds them with
        <span class="mn-code-inline">search_snippets</span>. Read-only here.
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading snippets…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load snippets: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      </div>

      <template v-else>
        <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
          no snippets yet — save one over MCP with
          <span class="mn-code-inline">save_snippet</span>.
        </p>

        <template v-else>
          <div class="filters">
            <select v-model="projectFilter" class="filter mn-mono-sm" aria-label="filter by project" data-test="project-filter">
              <option value="">all projects</option>
              <option v-for="p in projects" :key="p || '_global'" :value="p">{{ projectLabel(p) }}</option>
            </select>
            <select v-model="languageFilter" class="filter mn-mono-sm" aria-label="filter by language" data-test="language-filter">
              <option value="">all languages</option>
              <option v-for="l in languages" :key="l" :value="l">{{ l }}</option>
            </select>
            <select v-model="tagFilter" class="filter mn-mono-sm" aria-label="filter by tag" data-test="tag-filter">
              <option value="">all tags</option>
              <option v-for="t in tags" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>

          <section v-for="group in groups" :key="group.project || '_global'" class="group" data-test="group">
            <h2 class="group-title mn-h3">
              <span class="scope-prefix mn-mono-sm">project /</span> {{ projectLabel(group.project) }}
            </h2>
            <div class="cards">
              <SnippetCard
                v-for="s in group.snippets"
                :key="s.id"
                :snippet="s"
                :data-flash-id="s.id"
                :data-ref-id="s.public_id"
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
