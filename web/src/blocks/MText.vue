<script setup lang="ts">
import { computed } from 'vue'
import { renderInline } from '@/lib/markdown'

const props = defineProps<{ id?: string; content?: string }>()
const html = computed(() => renderInline(props.content))
</script>

<template>
  <p v-if="html" class="mn-body mn-md" v-html="html" />
</template>
