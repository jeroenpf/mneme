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
  session?: number | string
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

// session carries either a sitting number or a free-text note about how to run
// the phase. A bare number needs the word to mean anything; a note already
// names itself, and prefixing it reads as "session Clean session."
const sessionText = computed(() => {
  if (props.session == null) return ''
  const value = String(props.session).trim()
  return /^\d+$/.test(value) ? `session ${value}` : value
})
</script>

<template>
  <section :id="id" class="mn-anchor subphase">
    <div v-if="isTop" class="sec-head" :class="{ 'no-num': !num }">
      <span v-if="num" class="sec-num mn-mono-sm">{{ num }}</span>
      <h2 class="mn-h2 mn-md heading" v-html="titleHtml" />
      <BlockRefCopy :block-id="id" kind="block" />
      <span v-if="sessionText" class="session mn-mono-sm">{{ sessionText }}</span>
    </div>
    <div v-else class="sub-head" :class="{ 'no-num': !num }">
      <span v-if="num" class="sub-num mn-mono-sm">{{ num }}</span>
      <h3 class="mn-h3 mn-md heading" v-html="titleHtml" />
      <BlockRefCopy :block-id="id" kind="block" />
      <span v-if="sessionText" class="session mn-mono-sm">{{ sessionText }}</span>
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
/* Masthead grid: number gutter, heading, ref chip on the first row; the session
   note spans a second row under the heading. Inline, a note long enough to be
   worth writing starves the heading column and wraps the title into a stack. */
.sec-head,
.sub-head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: baseline;
  column-gap: var(--space-3);
  row-gap: var(--space-1);
}
/* Top-level phase: full masthead, identical to a numbered section. */
.sec-head {
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-strong);
}
/* Nested phase (under a parent section): a lighter subheading — smaller type,
   soft rule — so the parent section stays the dominant level. */
.sub-head {
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--border-soft);
}
/* Without a number there is no gutter to leave room for. */
.sec-head.no-num,
.sub-head.no-num {
  grid-template-columns: minmax(0, 1fr) auto;
}
.sec-num,
.sub-num {
  color: var(--eyebrow);
  font-weight: 500;
}
/* Hangs under the heading, clear of the number gutter, so the number keeps
   marking the whole masthead. */
.session {
  grid-row: 2;
  grid-column: 2 / -1;
}
.no-num .session {
  grid-column: 1 / -1;
}
.sec-head:hover :deep(.block-ref),
.sec-head:focus-within :deep(.block-ref),
.sub-head:hover :deep(.block-ref),
.sub-head:focus-within :deep(.block-ref) {
  opacity: 1;
}
</style>
