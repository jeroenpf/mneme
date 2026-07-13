<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { search, type SearchHit } from '@/api/search'

const route = useRoute()
const hits = ref<SearchHit[]>([])
const loading = ref(false)
const error = ref<Error | null>(null)

const q = computed(() => String(route.query.q ?? '').trim())

const grouped = computed(() => {
  const g: Record<string, SearchHit[]> = {}
  for (const h of hits.value) {
    ;(g[h.type] ??= []).push(h)
  }
  return g
})

async function run(query: string) {
  if (!query) {
    hits.value = []
    return
  }
  loading.value = true
  error.value = null
  try {
    hits.value = await search(query)
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
    hits.value = []
  } finally {
    loading.value = false
  }
}

watch(q, (query) => run(query), { immediate: true })

function linkFor(h: SearchHit): string | null {
  return h.type === 'documents' ? `/doc/${h.id}` : null
}

// ts_headline marks matches with <<…>>; render them as <mark> after escaping.
function renderExcerpt(raw: string): string {
  const esc = raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  return esc.replace(/&lt;&lt;(.+?)&gt;&gt;/g, '<mark>$1</mark>')
}
</script>

<template>
  <div>
    <header class="topbar">
      <RouterLink to="/" class="brand"><span class="glyph">⬡</span> mneme</RouterLink>
      <span class="mn-label">/ search</span>
      <RouterLink to="/" class="nav-link mn-mono-sm" data-test="to-registry">registry →</RouterLink>
    </header>

    <main class="content">
      <p class="mn-body-sm intro">
        Unified full-text search across documents, decisions, snippets, solutions, and journal.
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">searching…</p>
      <div v-else-if="error" class="error-box mn-body-sm" data-test="error">
        could not search: {{ error.message }}
      </div>

      <template v-else-if="q && hits.length">
        <section v-for="(items, type) in grouped" :key="type" class="group">
          <h2 class="mn-label group-head">{{ type }} <span class="count">{{ items.length }}</span></h2>
          <ul class="hits">
            <li v-for="h in items" :key="h.id" class="hit">
              <RouterLink v-if="linkFor(h)" :to="linkFor(h)!" class="hit-title">{{ h.title }}</RouterLink>
              <span v-else class="hit-title">{{ h.title }}</span>
              <span v-if="h.project" class="hit-project mn-mono-sm">{{ h.project }}</span>
              <p class="hit-excerpt mn-body-sm" v-html="renderExcerpt(h.excerpt)"></p>
            </li>
          </ul>
        </section>
      </template>

      <p v-else-if="q" class="mn-body-sm empty" data-test="no-results">no results for “{{ q }}”.</p>
      <p v-else class="mn-body-sm empty" data-test="empty">type a query in the top bar and press enter.</p>
    </main>
  </div>
</template>

<style scoped>
.topbar {
  position: sticky; top: 0; z-index: 10;
  display: flex; align-items: center; gap: var(--space-3);
  height: var(--topbar-height); padding: 0 var(--space-6);
  background: var(--bg); border-bottom: 1px solid var(--border);
}
.brand { font-family: var(--font-display); font-weight: 700; font-size: 16px; color: var(--text-primary); text-decoration: none; }
.glyph { color: var(--accent); }
.nav-link { margin-left: auto; color: var(--text-muted); text-decoration: none; }
.nav-link:hover { color: var(--accent); }
.content {
  max-width: var(--content-max); width: 100%; margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex; flex-direction: column; gap: var(--space-6); min-width: 0;
}
.intro { color: var(--text-muted); margin-top: calc(-1 * var(--space-2)); }
.group { display: flex; flex-direction: column; gap: var(--space-3); }
.group-head { text-transform: uppercase; }
.count { color: var(--text-muted); }
.hits { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-3); }
.hit {
  border: 1px solid var(--border); border-radius: var(--radius-md);
  padding: var(--space-4); background: var(--bg-surface);
  display: flex; flex-direction: column; gap: var(--space-2);
}
.hit-title { color: var(--text-primary); text-decoration: none; font-weight: 600; }
a.hit-title:hover { color: var(--accent); }
.hit-project { color: var(--text-muted); }
.hit-excerpt { color: var(--text-secondary); margin: 0; }
/* :deep() — the <mark> is injected via v-html, which scoped CSS can't
   reach without it (discovered during the Task 6 browser smoke; otherwise
   the browser's default yellow highlight wins over the accent styling). */
.hit-excerpt :deep(mark) { background: var(--accent-dim); color: var(--accent); }
.empty { color: var(--text-muted); }
.error-box {
  border: 1px solid var(--red-border); background: var(--red-dim);
  border-radius: var(--radius-md); padding: var(--space-4); color: var(--text-secondary);
}
</style>
