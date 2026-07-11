<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import { highlightCode } from '@/lib/highlight'

const props = defineProps<{ id?: string; lang?: string; filename?: string; content?: string }>()

const html = ref('')
const copied = ref(false)

watchEffect(async () => {
  html.value = await highlightCode(props.content ?? '', props.lang)
})

async function copy() {
  await navigator.clipboard.writeText(props.content ?? '')
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <figure class="code">
    <figcaption class="head">
      <span v-if="lang" class="mn-label" data-test="lang">{{ lang }}</span>
      <span v-if="filename" class="mn-mono-sm" data-test="filename">{{ filename }}</span>
      <button class="copy mn-mono-sm" data-test="copy" @click="copy">
        {{ copied ? 'copied' : 'copy' }}
      </button>
    </figcaption>
    <pre class="mn-mono"><code v-html="html" /></pre>
  </figure>
</template>

<style scoped>
.code {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  overflow: hidden;
  margin: 0;
}
.head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}
.copy {
  margin-left: auto;
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}
.copy:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}
pre {
  margin: 0;
  padding: var(--space-4);
  overflow-x: auto;
}

/* Prism token colors — semantic palette only. */
pre :deep(.token.comment),
pre :deep(.token.prolog),
pre :deep(.token.doctype),
pre :deep(.token.cdata) { color: var(--text-faint); }
pre :deep(.token.punctuation) { color: var(--text-muted); }
pre :deep(.token.operator),
pre :deep(.token.entity),
pre :deep(.token.url) { color: var(--text-secondary); }
pre :deep(.token.keyword),
pre :deep(.token.atrule),
pre :deep(.token.important) { color: var(--purple); }
pre :deep(.token.string),
pre :deep(.token.char),
pre :deep(.token.attr-value),
pre :deep(.token.regex) { color: var(--green); }
pre :deep(.token.number),
pre :deep(.token.boolean),
pre :deep(.token.constant),
pre :deep(.token.builtin),
pre :deep(.token.symbol) { color: var(--yellow); }
pre :deep(.token.function),
pre :deep(.token.class-name),
pre :deep(.token.selector),
pre :deep(.token.attr-name),
pre :deep(.token.property) { color: var(--blue); }
pre :deep(.token.tag),
pre :deep(.token.variable),
pre :deep(.token.deleted) { color: var(--red); }
</style>
