<script setup lang="ts">
import { computed, nextTick, toRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BlockRenderer from '@/blocks/BlockRenderer.vue'
import MetaHeader from '@/components/MetaHeader.vue'
import PhaseTracker from '@/components/PhaseTracker.vue'
import SectionNav from '@/components/SectionNav.vue'
import { useDocument } from '@/composables/useDocument'
import { phasesFromMeta } from '@/lib/phases'
import { sectionNavItems } from '@/lib/toc'

const props = defineProps<{ id: string }>()

const route = useRoute()
const router = useRouter()
const { doc, loading, error, refresh } = useDocument(toRef(props, 'id'))

const phases = computed(() => phasesFromMeta(doc.value?.meta))
const navItems = computed(() => sectionNavItems(doc.value?.body))
const blocks = computed(() => {
  const sections = doc.value?.body?.sections
  return Array.isArray(sections) ? (sections as Array<Record<string, unknown>>) : []
})

// Deep links: scroll once content exists (initial load), then follow
// hash changes from the section nav.
function scrollToHash(smooth: boolean) {
  const hash = route.hash.slice(1)
  if (!hash) return
  document.getElementById(hash)?.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto' })
}

watch(loading, async (isLoading) => {
  if (isLoading) return
  await nextTick()
  scrollToHash(false)
})

watch(
  () => route.hash,
  async () => {
    await nextTick()
    scrollToHash(true)
  },
)

// Back returns to the registry with its filter query intact when we got
// here from it; a cold deep link falls back to the bare registry.
function goBack() {
  if (window.history.state?.back) router.back()
  else void router.push('/')
}
</script>

<template>
  <div>
    <header class="topbar">
      <button class="back mn-mono-sm" data-test="back" @click="goBack">←</button>
      <RouterLink to="/" class="brand"><span class="glyph">⬡</span> mneme</RouterLink>
      <span class="mn-label">/ doc /</span>
      <span class="mn-mono-sm">{{ id }}</span>
      <span v-if="doc" class="status mn-mono-sm" data-test="doc-status">
        <span class="status-dot" :class="`status-${doc.status}`" />
        {{ doc.status }}
      </span>
    </header>

    <p v-if="loading" class="mn-mono-sm py-8 text-center">loading document…</p>

    <div v-else-if="error" class="error mn-body-sm mx-auto my-8 max-w-lg" data-test="doc-error">
      <p>could not load document: {{ error.message }}</p>
      <button class="retry mn-mono-sm" @click="refresh">retry</button>
      <RouterLink to="/" class="link mn-mono-sm">back to registry</RouterLink>
    </div>

    <div v-else-if="doc" class="layout">
      <aside v-if="phases.length || navItems.length" class="sidebar">
        <PhaseTracker v-if="phases.length" :phases="phases" />
        <SectionNav v-if="navItems.length" :items="navItems" />
      </aside>

      <main class="content">
        <MetaHeader :doc="doc" />
        <BlockRenderer :blocks="blocks" />
      </main>
    </div>
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
.back {
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}
.back:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
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
.status {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: var(--radius-xs);
}
.status-todo        { background: var(--status-todo); }
.status-in-progress { background: var(--status-wip); }
.status-complete    { background: var(--status-done); }
.status-blocked     { background: var(--status-blocked); }
.status-archived    { background: var(--status-archived); }

.layout {
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
}
.sidebar {
  position: sticky;
  top: var(--topbar-height);
  align-self: start;
  max-height: calc(100vh - var(--topbar-height));
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  padding: var(--space-6);
  border-right: 1px solid var(--border-soft);
}
.content {
  max-width: var(--content-max);
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  min-width: 0;
}

.error {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.retry {
  margin: var(--space-2) var(--space-2) 0 0;
  padding: 4px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}
.link {
  color: var(--accent);
  text-decoration: none;
}
</style>
