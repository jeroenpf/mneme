<script setup lang="ts">
import { renderInline } from '@/lib/markdown'
import BlockRefCopy from './BlockRefCopy.vue'
import type { TaskItem } from '@/types'

defineProps<{ id?: string; title?: string; tasks?: TaskItem[] }>()
</script>

<template>
  <div v-if="tasks?.length">
    <h4 v-if="title" class="mn-label mb-2">{{ title }}</h4>
    <ul class="tasks">
      <li v-for="task in tasks" :id="task.id" :key="task.id" class="mn-anchor task">
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
        <BlockRefCopy :block-id="task.id" kind="task" class="task-ref" />
      </li>
    </ul>
  </div>
</template>

<style scoped>
.tasks {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  list-style: none;
  padding: 0;
  margin: 0;
}
.task {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-1) var(--space-2);
  margin-left: calc(-1 * var(--space-2));
  border-radius: var(--radius-sm);
  transition: background var(--duration-fast) var(--ease-out);
}
.task:hover {
  background: var(--bg-hover);
}
.task-ref {
  margin-left: auto;
  align-self: center;
}
.task:hover :deep(.block-ref),
.task:focus-within :deep(.block-ref) {
  opacity: 1;
}
/* Rounded-square checkbox — an empty box for todo, a filled green box with an
   inverse check for done, matching the phase spine's "done is green" language.
   (The task model carries only a `done` boolean — no separate wip state.) */
.box {
  flex: none;
  width: 17px;
  height: 17px;
  margin-top: 1px;
  display: grid;
  place-items: center;
  border: 1.5px solid var(--border-strong);
  border-radius: 5px;
  font-family: var(--font-mono);
  font-size: 10px;
  line-height: 1;
  color: transparent;
}
.box.done {
  background: var(--green);
  border-color: var(--green);
  /* light check in the light themes, dark in Ink — always the node's inverse */
  color: var(--bg-elevated);
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
