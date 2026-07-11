<script setup lang="ts">
import { computed } from 'vue'
import type { Document } from '@/types'
import { renderInline } from '@/lib/markdown'
import MKeyValue from '@/blocks/MKeyValue.vue'

const props = defineProps<{ doc: Document }>()

const titleHtml = computed(() => renderInline(props.doc.title))

const description = computed(() => {
  const d = props.doc.meta?.description
  return typeof d === 'string' ? d : ''
})

const customFields = computed(() => {
  const cf = props.doc.meta?.custom_fields
  if (typeof cf !== 'object' || cf === null || Array.isArray(cf)) return undefined
  const entries = Object.entries(cf).filter(([, v]) => typeof v === 'string')
  return entries.length ? (Object.fromEntries(entries) as Record<string, string>) : undefined
})

const day = (iso: string) => iso.slice(0, 10)

const cells = computed(() => {
  const d = props.doc
  const out: Array<{ label: string; value: string }> = []
  if (d.project) out.push({ label: 'project', value: d.project })
  if (d.ticket) out.push({ label: 'ticket', value: d.ticket })
  if (d.phase_current != null && d.phase_total != null)
    out.push({ label: 'phase', value: `${d.phase_current}/${d.phase_total}` })
  out.push({ label: 'created', value: day(d.created_at) })
  out.push({ label: 'updated', value: day(d.updated_at) })
  return out
})
</script>

<template>
  <header class="meta-header">
    <div class="flex flex-wrap items-baseline gap-3">
      <span class="mn-label">{{ doc.type }}</span>
      <span v-if="doc.ticket" class="mn-mono-sm">{{ doc.ticket }}</span>
      <span v-if="doc.repo" class="mn-mono-sm text-text-faint">⎇ {{ doc.repo }}</span>
      <span v-for="tag in doc.tags" :key="tag" class="mn-mono-sm">#{{ tag }}</span>
    </div>

    <h1 class="mn-display mn-md" v-html="titleHtml" />

    <p v-if="description" class="mn-body">{{ description }}</p>

    <div class="grid-strip">
      <div v-for="cell in cells" :key="cell.label" class="cell" data-test="meta-cell">
        <div class="mn-label">{{ cell.label }}</div>
        <div class="mn-mono">{{ cell.value }}</div>
      </div>
    </div>

    <MKeyValue v-if="customFields" :data="customFields" />
  </header>
</template>

<style scoped>
.meta-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding-bottom: var(--space-6);
  border-bottom: 1px solid var(--border);
}
.grid-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
}
.cell + .cell {
  border-left: 1px solid var(--border-soft);
}
</style>
