<script setup lang="ts">
import { computed, inject, provide } from 'vue'
import BlockRenderer from './BlockRenderer.vue'
import BlockRefCopy from './BlockRefCopy.vue'
import MTaskList from './MTaskList.vue'
import { renderInline, renderParagraphs } from '@/lib/markdown'
import type { TaskItem } from '@/types'

const props = defineProps<{
  id?: string
  num?: string
  title?: string
  session?: number
  description?: string
  tasks?: TaskItem[]
  children?: unknown[]
}>()

// A subphase is a phase-flavored section, so it mirrors MSection's depth
// handling: the numbered masthead (mn-h2 + strong rule) at the top level, a
// lighter subheading (mn-h3 + soft rule) when nested inside a parent section —
// e.g. individual phases under an "Implementation phases" section, which must
// read as subordinate to it, not at the same weight. Depth is injected so the
// block JSON stays depth-free.
const depth = inject<number>('mn-section-depth', 0)
provide('mn-section-depth', depth + 1)
const isTop = computed(() => depth === 0)

const titleHtml = computed(() => renderInline(props.title))
const descParagraphs = computed(() => renderParagraphs(props.description))
</script>

<template>
  <section :id="id" class="mn-anchor subphase">
    <div v-if="isTop" class="sec-head">
      <span v-if="num" class="sec-num mn-mono-sm">{{ num }}</span>
      <h2 class="mn-h2 mn-md heading" v-html="titleHtml" />
      <span v-if="session != null" class="mn-label session">session {{ session }}</span>
      <BlockRefCopy :block-id="id" kind="block" />
    </div>
    <div v-else class="sub-head">
      <span v-if="num" class="sub-num mn-mono-sm">{{ num }}</span>
      <h3 class="mn-h3 mn-md heading" v-html="titleHtml" />
      <span v-if="session != null" class="mn-label session">session {{ session }}</span>
      <BlockRefCopy :block-id="id" kind="block" />
    </div>
    <div v-if="descParagraphs.length" class="mn-prose">
      <p v-for="(para, i) in descParagraphs" :key="i" class="mn-body mn-md" v-html="para" />
    </div>
    <MTaskList v-if="tasks?.length" :tasks="tasks" />
    <BlockRenderer
      v-if="children?.length"
      :blocks="children as Array<Record<string, unknown>>"
    />
  </section>
</template>

<style scoped>
/* A subphase renders like a numbered section — a phase is just a section that
   carries tasks, so its checkboxes sit flat in the reading column rather than
   boxed inside a card. Its phase progress lives in the sidebar phase spine. */
.subphase {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
/* Top-level phase: full masthead, identical to a numbered section. */
.sec-head {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-strong);
}
/* Nested phase (under a parent section): a lighter subheading — smaller type,
   soft rule — so the parent section stays the dominant level. */
.sub-head {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--border-soft);
}
.sec-num,
.sub-num {
  flex: none;
  color: var(--eyebrow);
  font-weight: 500;
}
.heading {
  flex: 1;
}
.session {
  flex: none;
}
.sec-head:hover :deep(.block-ref),
.sec-head:focus-within :deep(.block-ref),
.sub-head:hover :deep(.block-ref),
.sub-head:focus-within :deep(.block-ref) {
  opacity: 1;
}
</style>
