<script setup lang="ts">
import { typeToComponent } from './index'

defineProps<{ blocks: Array<Record<string, unknown>> }>()
</script>

<template>
  <component
    v-for="(block, i) in blocks"
    :is="typeToComponent((block as { type?: string }).type)"
    :key="(block as { id?: string }).id ?? i"
    v-bind="block"
  />
</template>
