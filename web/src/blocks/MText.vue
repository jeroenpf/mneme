<script setup lang="ts">
defineProps<{ id?: string; content?: string }>()
</script>

<template>
  <p class="mn-body">{{ content }}</p>
</template>
