<script setup lang="ts">
import RefChip from '@/components/RefChip.vue'
import { useDocPublicId } from '@/composables/useDocRef'

// A hover-revealed "copy reference" control for a block or task inside the
// document viewer. Renders only when both the owning document's public id (from
// provide/inject) and the block/task id are known, so it stays out of the way
// of the reading experience and never emits a broken reference.
defineProps<{ blockId?: string; kind: 'block' | 'task' }>()

const docId = useDocPublicId()
</script>

<template>
  <RefChip
    v-if="docId && blockId"
    class="block-ref"
    compact
    :kind="kind"
    :public-id="blockId"
    :doc-id="docId"
  />
</template>

<style scoped>
/* Subtle by default; the containing block reveals it on hover/focus-within
   (see MSection/MSubphase/MTaskList). Kept in flow so it never shifts layout. */
.block-ref {
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-out);
}
.block-ref:focus-within {
  opacity: 1;
}
@media (prefers-reduced-motion: reduce) {
  .block-ref {
    transition: none;
  }
}
</style>
