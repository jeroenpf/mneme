<script setup lang="ts">
interface TaskItem { id: string; title: string; done?: boolean; content?: string }
defineProps<{ id?: string; tasks?: TaskItem[] }>()
</script>

<template>
  <ul class="space-y-1">
    <li v-for="task in tasks ?? []" :key="task.id" class="mn-body-sm">
      <span class="mn-mono-sm">{{ task.done ? '[x]' : '[ ]' }}</span>
      {{ task.title }}
    </li>
  </ul>
</template>
