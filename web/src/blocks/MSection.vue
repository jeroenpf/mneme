<script setup lang="ts">
import { computed, inject, provide } from 'vue'
import BlockRenderer from './BlockRenderer.vue'
import { renderInline } from '@/lib/markdown'

const props = defineProps<{ id?: string; title?: string; content?: string; children?: unknown[] }>()

// Nesting depth drives heading size: top-level sections are h2/mn-h2,
// anything deeper is h3/mn-h3. Injected rather than passed as a prop so
// the block JSON stays depth-free.
const depth = inject<number>('mn-section-depth', 0)
provide('mn-section-depth', depth + 1)

const tag = computed(() => (depth === 0 ? 'h2' : 'h3'))
const cls = computed(() => (depth === 0 ? 'mn-h2' : 'mn-h3'))
const html = computed(() => renderInline(props.title))
// A section may carry its prose directly as `content` (the ergonomic
// channel that update_section patches) as well as nested `children`.
// Render both, mirroring MText so a section content string and a text
// block look identical.
const contentHtml = computed(() => renderInline(props.content))
</script>

<template>
  <section :id="id" class="mn-anchor m-section">
    <component :is="tag" v-if="html" :class="cls" class="mn-md heading" v-html="html" />
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
.heading {
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--border-soft);
}
</style>
