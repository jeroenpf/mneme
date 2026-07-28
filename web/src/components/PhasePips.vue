<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ current: number; total: number }>()

// current is the phase in flight (1-based): phases below it are done,
// it is the active segment, the rest are todo.
const pips = computed(() =>
  Array.from({ length: props.total }, (_, i) =>
    i + 1 < props.current ? 'done' : i + 1 === props.current ? 'wip' : 'todo',
  ),
)
</script>

<template>
  <div class="flex items-center gap-1" data-test="pips">
    <span v-for="(p, i) in pips" :key="i" class="pip" :class="`pip-${p}`" />
    <span class="mn-mono-sm ml-1">{{ current }}/{{ total }}</span>
  </div>
</template>

<style scoped>
.pip {
  width: 14px;
  height: 4px;
  border-radius: 1px;
  background: var(--bg-overlay);
}
.pip-done { background: var(--accent); }
.pip-wip  { background: var(--accent-dim); box-shadow: inset 0 0 0 1px var(--accent-border); }
</style>
