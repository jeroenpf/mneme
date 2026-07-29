<script setup lang="ts">
import { computed, nextTick, toRef, watch } from 'vue'
import { useRoute } from 'vue-router'
import BlockRenderer from '@/blocks/BlockRenderer.vue'
import DocHistory from '@/components/DocHistory.vue'
import MetaHeader from '@/components/MetaHeader.vue'
import RelatedPanel from '@/components/RelatedPanel.vue'
import PhaseTracker from '@/components/PhaseTracker.vue'
import SectionNav from '@/components/SectionNav.vue'
import { useDocument } from '@/composables/useDocument'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import { provideDocPublicId } from '@/composables/useDocRef'
import { phasesFromMeta } from '@/lib/phases'
import { sectionNavItems } from '@/lib/toc'
import { flashElement } from '@/lib/flash'

const props = defineProps<{ id: string }>()

const route = useRoute()
const { doc, loading, error, refresh } = useDocument(toRef(props, 'id'))

// Expose this document's public id to the block renderers so block/task copy
// controls can build nested mneme:// references.
provideDocPublicId(computed(() => doc.value?.public_id))

// Live updates: quietly refetch this document when the agent edits it over
// MCP. A *silent* refresh swaps the doc in place without toggling `loading`,
// so the content isn't torn down and rebuilt — the reader's scroll position
// survives and (from P3) the changed block can flash. blockId is empty until
// P3, so this is refresh-only for now.
useLiveRefresh('documents', {
  refresh: () => refresh({ silent: true }),
  match: (ev) => ev.id === props.id,
  flashTarget: (ev) => (ev.blockId ? `#${ev.blockId}` : null),
})

const phases = computed(() => phasesFromMeta(doc.value?.meta))
const navItems = computed(() => sectionNavItems(doc.value?.body))
const blocks = computed(() => {
  const sections = doc.value?.body?.sections
  if (!Array.isArray(sections)) return []
  // The TOC numbers every top-level entry 01…NN. Mirror that number onto the
  // body's section AND subphase blocks so their masthead marker matches the
  // nav — a subphase renders as a numbered section, not a badged card.
  const numById = new Map(navItems.value.map((i) => [i.id, i.num]))
  return (sections as Array<Record<string, unknown>>).map((b) =>
    (b?.type === 'section' || b?.type === 'subphase') &&
    typeof b.id === 'string' &&
    numById.has(b.id)
      ? { ...b, num: numById.get(b.id) }
      : b,
  )
})

// Deep links: scroll once content exists (initial load), then follow hash
// changes from the section nav. A pasted block/task reference lands here as
// #<blockId>, so flash the target too — the same highlight live edits use — to
// select it, not just scroll to it.
function scrollToHash(smooth: boolean) {
  const hash = route.hash.slice(1)
  if (!hash) return
  const el = document.getElementById(hash)
  if (!el) return
  el.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto', block: 'start' })
  flashElement(el)
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
</script>

<template>
  <div>
    <p v-if="loading" class="mn-mono-sm py-8 text-center">loading document…</p>

    <div v-else-if="error" class="error mn-body-sm mx-auto my-8 max-w-lg" data-test="doc-error">
      <p>could not load document: {{ error.message }}</p>
      <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      <RouterLink to="/" class="link mn-mono-sm">back to registry</RouterLink>
    </div>

    <div v-else-if="doc" class="layout">
      <aside v-if="phases.length || navItems.length" class="sidebar">
        <PhaseTracker v-if="phases.length" :phases="phases" />
        <SectionNav v-if="navItems.length" :items="navItems" />
      </aside>

      <main class="content">
        <div class="head-group">
          <MetaHeader :doc="doc" />
          <DocHistory :doc-id="doc.id" :current-revision="doc.revision" @restored="refresh({ silent: true })" />
        </div>
        <BlockRenderer :blocks="blocks" />
        <RelatedPanel :doc-id="doc.id" :revision="doc.revision" />
      </main>
    </div>
  </div>
</template>

<style scoped>
.layout {
  display: grid;
  grid-template-columns: var(--sidebar-width) minmax(0, 1fr);
}
.sidebar {
  position: sticky;
  top: 0;
  align-self: start;
  max-height: 100vh;
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
  /* Generous separation between top-level sections — with the larger type,
     the old 32px read as cramped. Sections group at ~3× their internal gap. */
  gap: var(--space-12);
  min-width: 0;
}
/* Keep the history bar tight under the masthead; the big content gap applies
   between this group and the body blocks. */
.head-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
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
