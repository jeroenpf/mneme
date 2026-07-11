<script setup lang="ts">
import { ref, watchEffect } from 'vue'
import { renderDiagram } from '@/lib/mermaid'

const props = defineProps<{ id?: string; title?: string; content?: string }>()

const svg = ref('')
const failed = ref(false)

// mermaid.render needs a document-unique element id; block ids repeat
// across route visits, so suffix a module counter.
let seq = 0

watchEffect(async () => {
  if (!props.content) return
  failed.value = false
  try {
    svg.value = await renderDiagram(
      `mmd-${(props.id ?? 'd').replace(/[^\w-]/g, '')}-${seq++}`,
      props.content,
    )
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <figure class="diagram">
    <figcaption v-if="title" class="head mn-label">{{ title }}</figcaption>
    <div v-if="!failed && svg" class="body" v-html="svg" />
    <div v-else-if="failed" class="body err-body">
      <p class="mn-mono-sm err" data-test="diagram-error">diagram failed to render — source below</p>
      <pre class="mn-mono-sm src">{{ content }}</pre>
    </div>
  </figure>
</template>

<style scoped>
.diagram {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  overflow: hidden;
  margin: 0;
}
.head {
  padding: var(--space-2) var(--space-4);
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border);
}
.body {
  padding: var(--space-4);
  display: flex;
  justify-content: center;
  overflow-x: auto;
}
.body :deep(svg) {
  max-width: 100%;
  height: auto;
}
.err-body {
  flex-direction: column;
  justify-content: flex-start;
}
.err {
  color: var(--red);
}
.src {
  margin: var(--space-2) 0 0;
  color: var(--text-muted);
  white-space: pre-wrap;
}
</style>
