<script setup lang="ts">
import { computed } from 'vue'
import { renderParagraphs } from '@/lib/markdown'

const props = defineProps<{ id?: string; content?: string }>()
const paragraphs = computed(() => renderParagraphs(props.content))
</script>

<template>
  <div v-if="paragraphs.length" class="mn-prose">
    <p v-for="(html, i) in paragraphs" :key="i" class="mn-body mn-md" v-html="html" />
  </div>
</template>
