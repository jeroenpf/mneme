<script setup lang="ts">
import { computed, ref } from 'vue'
import { groupSolutions, useSolutions } from '@/composables/useSolutions'
import SolutionCard from '@/components/SolutionCard.vue'

const { items, loading, error, refresh } = useSolutions()

const projectFilter = ref('') // '' = all
const tagFilter = ref('') // '' = all

const filtered = computed(() =>
  items.value.filter(
    (s) =>
      (projectFilter.value === '' || (s.project ?? '') === projectFilter.value) &&
      (tagFilter.value === '' || s.tags.includes(tagFilter.value)),
  ),
)
const groups = computed(() => groupSolutions(filtered.value))
const isEmpty = computed(() => items.value.length === 0)

const projects = computed(() => {
  const set = new Set<string>()
  for (const s of items.value) set.add(s.project ?? '')
  return [...set].sort((a, b) => (a === '' ? -1 : b === '' ? 1 : a.localeCompare(b)))
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
    <header class="topbar">
      <RouterLink to="/" class="brand"><span class="glyph">⬡</span> mneme</RouterLink>
      <span class="mn-label">/ solutions</span>
      <RouterLink to="/" class="nav-link mn-mono-sm" data-test="to-registry">registry →</RouterLink>
      <RouterLink to="/snippets" class="nav-link-tight mn-mono-sm" data-test="to-snippets">snippets →</RouterLink>
      <RouterLink to="/journal" class="nav-link-tight mn-mono-sm" data-test="to-journal">journal →</RouterLink>
    </header>

    <main class="content">
      <p class="mn-body-sm intro">
        The error / solution database — recurring gotchas and the fixes that worked. Claude Code writes it with
        <span class="mn-code-inline">log_solution</span> and searches it with
        <span class="mn-code-inline">find_solution</span> before debugging. Read-only here.
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading solutions…</p>

      <div v-else-if="error" class="error-box mn-body-sm" data-test="error">
        <p>could not load solutions: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh">retry</button>
      </div>

      <template v-else>
        <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
          no solutions yet — log one over MCP with
          <span class="mn-code-inline">log_solution</span>.
        </p>

        <template v-else>
          <div class="filters">
            <select v-model="projectFilter" class="filter mn-mono-sm" aria-label="filter by project" data-test="project-filter">
              <option value="">all projects</option>
              <option v-for="p in projects" :key="p || '_global'" :value="p">{{ projectLabel(p) }}</option>
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
              <SolutionCard v-for="s in group.solutions" :key="s.id" :solution="s" />
            </div>
          </section>
        </template>
      </template>
    </main>
  </div>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  height: var(--topbar-height);
  padding: 0 var(--space-6);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
}
.brand {
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 16px;
  color: var(--text-primary);
  text-decoration: none;
}
.glyph {
  color: var(--accent);
}
.nav-link {
  margin-left: auto;
  color: var(--text-muted);
  text-decoration: none;
}
.nav-link-tight {
  color: var(--text-muted);
  text-decoration: none;
}
.nav-link:hover,
.nav-link-tight:hover {
  color: var(--accent);
}

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
.error-box {
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
