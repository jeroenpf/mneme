<script setup lang="ts">
import BlockRenderer from './BlockRenderer.vue'
import MTaskList from './MTaskList.vue'

interface TaskItem { id: string; title: string; done?: boolean }
defineProps<{
  id?: string
  num?: string
  title?: string
  session?: number
  description?: string
  tasks?: TaskItem[]
  children?: unknown[]
}>()
</script>

<template>
  <section class="space-y-3 border-l-2 border-border-soft pl-4">
    <header>
      <span class="mn-label">{{ num }} · session {{ session }}</span>
      <h2 class="mn-h2">{{ title }}</h2>
      <p v-if="description" class="mn-body-sm">{{ description }}</p>
    </header>
    <MTaskList v-if="tasks" :tasks="tasks" />
    <BlockRenderer v-if="children" :blocks="(children as Array<Record<string, unknown>>)" />
  </section>
</template>
