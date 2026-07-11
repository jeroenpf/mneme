<script setup lang="ts">
import { renderInline } from '@/lib/markdown'
import type { TaskItem } from '@/types'

defineProps<{ id?: string; title?: string; tasks?: TaskItem[] }>()
</script>

<template>
  <div v-if="tasks?.length">
    <h4 v-if="title" class="mn-label mb-2">{{ title }}</h4>
    <ul class="tasks">
      <li v-for="task in tasks" :key="task.id" class="task">
        <span class="box" :class="{ done: task.done }" aria-hidden="true">{{
          task.done ? '✓' : ''
        }}</span>
        <div class="min-w-0">
          <div class="flex flex-wrap items-baseline gap-2">
            <span
              class="mn-body-sm mn-md title"
              :class="{ 'title-done': task.done }"
              v-html="renderInline(task.title)"
            />
            <span v-for="tag in task.tags ?? []" :key="tag" class="mn-mono-sm">#{{ tag }}</span>
          </div>
          <div
            v-if="task.content"
            class="mn-body-sm mn-md content"
            v-html="renderInline(task.content)"
          />
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.tasks {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  list-style: none;
  padding: 0;
  margin: 0;
}
.task {
  display: flex;
  gap: var(--space-3);
}
.box {
  flex: none;
  width: 13px;
  height: 13px;
  margin-top: 3px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-size: 9px;
  line-height: 11px;
  text-align: center;
  color: var(--accent);
}
.box.done {
  background: var(--accent-dim);
  border-color: var(--accent-border);
}
.title,
.title :deep(*) {
  color: var(--text-primary);
}
.title-done,
.title-done :deep(*) {
  color: var(--text-muted);
}
.content {
  color: var(--text-muted);
}
</style>
