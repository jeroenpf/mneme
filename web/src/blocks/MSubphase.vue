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
    <div class="sec-head">
      <span v-if="num" class="sec-num mn-mono-sm">{{ num }}</span>
      <h2 class="mn-h2 mn-md heading" v-html="titleHtml" />
      <span v-if="session != null" class="mn-label session">session {{ session }}</span>
    </div>
    <p v-if="descHtml" class="mn-body mn-md" v-html="descHtml" />
    <MTaskList v-if="tasks?.length" :tasks="tasks" />
    <BlockRenderer
      v-if="children?.length"
      :blocks="children as Array<Record<string, unknown>>"
    />
  </section>
</template>

<style scoped>
/* A subphase renders exactly like a numbered section — a phase is just a
   section that carries tasks, so its checkboxes sit flat in the reading
   column rather than boxed inside a card. Its phase progress lives in the
   sidebar phase spine, not here. */
.subphase {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
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
.session {
  flex: none;
}
</style>
