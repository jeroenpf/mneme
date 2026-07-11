<script setup lang="ts">
import { computed } from 'vue'
import BlockRenderer from './BlockRenderer.vue'
import MTaskList from './MTaskList.vue'
import { renderInline } from '@/lib/markdown'
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

const titleHtml = computed(() => renderInline(props.title))
const descHtml = computed(() => renderInline(props.description))
</script>

<template>
  <section :id="id" class="mn-anchor subphase">
    <header class="head">
      <span v-if="num" class="num mn-mono-sm">{{ num }}</span>
      <h2 class="mn-h2 mn-md" v-html="titleHtml" />
      <span v-if="session != null" class="mn-label ml-auto">session {{ session }}</span>
    </header>
    <p v-if="descHtml" class="mn-body-sm mn-md" v-html="descHtml" />
    <MTaskList v-if="tasks?.length" :tasks="tasks" />
    <BlockRenderer
      v-if="children?.length"
      :blocks="children as Array<Record<string, unknown>>"
    />
  </section>
</template>

<style scoped>
.subphase {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--border);
  border-left: 2px solid var(--accent-border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
}
.head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.num {
  flex: none;
  padding: 2px 7px;
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-sm);
  background: var(--accent-dim);
  color: var(--accent);
}
</style>
