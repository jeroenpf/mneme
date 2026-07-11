<script setup lang="ts">
import { computed } from 'vue'
import { renderInline } from '@/lib/markdown'

const props = defineProps<{ id?: string; variant?: string; title?: string; content?: string }>()

const GLYPHS: Record<string, string> = {
  info: 'ℹ',
  warn: '▲',
  success: '✓',
  danger: '✕',
  note: '◈',
}

const variant = computed(() => (props.variant && props.variant in GLYPHS ? props.variant : 'note'))
const html = computed(() => renderInline(props.content))
</script>

<template>
  <aside class="callout" :class="`callout-${variant}`">
    <span class="glyph mn-mono" aria-hidden="true">{{ GLYPHS[variant] }}</span>
    <div class="min-w-0">
      <strong v-if="title" class="mn-label title">{{ title }}</strong>
      <div class="mn-body-sm mn-md" v-html="html" />
    </div>
  </aside>
</template>

<style scoped>
.callout {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
}
.glyph { flex: none; }
.title { display: block; margin-bottom: var(--space-1); }

.callout-info    { background: var(--blue-dim);   border-color: var(--blue-border); }
.callout-info    .glyph, .callout-info    .title { color: var(--blue); }
.callout-warn    { background: var(--yellow-dim); border-color: var(--yellow-border); }
.callout-warn    .glyph, .callout-warn    .title { color: var(--yellow); }
.callout-success { background: var(--green-dim);  border-color: var(--green-border); }
.callout-success .glyph, .callout-success .title { color: var(--green); }
.callout-danger  { background: var(--red-dim);    border-color: var(--red-border); }
.callout-danger  .glyph, .callout-danger  .title { color: var(--red); }
.callout-note    { background: var(--purple-dim); border-color: var(--purple-border); }
.callout-note    .glyph, .callout-note    .title { color: var(--purple); }
</style>
