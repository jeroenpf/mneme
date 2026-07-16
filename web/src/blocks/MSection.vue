<script setup lang="ts">
import { computed, inject, provide } from 'vue'
import BlockRenderer from './BlockRenderer.vue'
import { renderInline } from '@/lib/markdown'

const props = defineProps<{
  id?: string
  num?: string
  title?: string
  content?: string
  children?: unknown[]
}>()

// Nesting depth drives heading treatment: top-level sections get the numbered
// masthead-style header (mn-h2 + sec-num + rule); anything deeper is a plain
// mn-h3 subheading. Injected rather than passed as a prop so the block JSON
// stays depth-free.
const depth = inject<number>('mn-section-depth', 0)
provide('mn-section-depth', depth + 1)

const isTop = computed(() => depth === 0)
const html = computed(() => renderInline(props.title))
// A section may carry its prose directly as `content` (the ergonomic
// channel that update_section patches) as well as nested `children`.
// Render both, mirroring MText so a section content string and a text
// block look identical.
const contentHtml = computed(() => renderInline(props.content))
</script>

<template>
  <section :id="id" class="mn-anchor m-section">
    <div v-if="html && isTop" class="sec-head">
      <span v-if="num" class="sec-num mn-mono-sm">{{ num }}</span>
      <h2 class="mn-h2 mn-md heading" v-html="html" />
    </div>
    <h3 v-else-if="html" class="mn-h3 mn-md subheading" v-html="html" />
    <p v-if="contentHtml" class="mn-body mn-md" v-html="contentHtml" />
    <BlockRenderer :blocks="(children ?? []) as Array<Record<string, unknown>>" />
  </section>
</template>

<style scoped>
.m-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
/* Top-level section masthead: amber sequential number + heading, over a rule
   strong enough to actually register as a section divider. */
.sec-head {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-strong);
}
.sec-num {
  flex: none;
  color: var(--eyebrow);
  font-weight: 500;
}
.heading {
  flex: 1;
}
</style>
