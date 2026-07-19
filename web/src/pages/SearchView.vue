<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { search, searchStatus, type SearchHit, type SearchStatus } from '@/api/search'
import { tryOpenRef } from '@/lib/openRef'

const route = useRoute()
const router = useRouter()
const hits = ref<SearchHit[]>([])
const loading = ref(false)
const error = ref<Error | null>(null)

const q = computed(() => String(route.query.q ?? '').trim())

// Embedding coverage line — non-blocking. Search works regardless of whether
// this resolves; a failure just hides the line.
const status = ref<SearchStatus | null>(null)
onMounted(async () => {
  try {
    status.value = await searchStatus()
  } catch {
    /* non-blocking */
  }
})
const coverage = computed(() => {
  const s = status.value
  if (!s) return ''
  if (!s.enabled) return 'FTS-only — no embedding key'
  const e = s.items.reduce((n, i) => n + i.embedded, 0)
  const t = s.items.reduce((n, i) => n + i.total, 0)
  return `semantic: ${e}/${t} embedded`
})

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

// A mneme:// reference (or bare public id) that lands in ?q= opens its target
// instead of running a full-text search for the literal reference string.
watch(
  q,
  (query) => {
    if (query && tryOpenRef(router, query)) return
    run(query)
  },
  { immediate: true },
)

// Deep-link every result type to its actual viewer with the entity flagged
// for highlight. Documents open their own page; the list-backed types open
// their list with ?flash=<id>, which useDeepLinkFlash scrolls to and flashes.
// Memory rows flash on their key (the hit title), not the uuid.
function linkFor(h: SearchHit): string | null {
  switch (h.type) {
    case 'documents':
      return `/doc/${h.id}`
    case 'decisions':
      return `/decisions?flash=${encodeURIComponent(h.id)}`
    case 'snippets':
      return `/snippets?flash=${encodeURIComponent(h.id)}`
    case 'solutions':
      return `/solutions?flash=${encodeURIComponent(h.id)}`
    case 'journal':
      return `/journal?flash=${encodeURIComponent(h.id)}`
    case 'memory':
      return `/memory?flash=${encodeURIComponent(h.title)}`
    default:
      return null
  }
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
    <main class="content">
      <p class="mn-body-sm intro">
        Unified search across documents, decisions, snippets, solutions, journal, and memory.
      </p>
      <p v-if="coverage" class="mn-mono-sm coverage" data-test="coverage">{{ coverage }}</p>

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
      <p v-else class="mn-body-sm empty" data-test="empty">type a query in the rail search and press enter.</p>
    </main>
  </div>
</template>

<style scoped>
.content {
  max-width: var(--content-max); width: 100%; margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex; flex-direction: column; gap: var(--space-6); min-width: 0;
}
.intro { color: var(--text-muted); margin-top: calc(-1 * var(--space-2)); }
.coverage { color: var(--text-faint); margin-top: calc(-1 * var(--space-4)); }
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
