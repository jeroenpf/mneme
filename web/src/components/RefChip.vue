<script setup lang="ts">
import { computed } from 'vue'
import { copyText } from '@/lib/clipboard'
import { formatRef, type Kind } from '@/lib/mnemeRef'
import { useToast } from '@/composables/useToast'

// A compact public-id chip with two copy actions: the id text itself copies the
// bare public id; the adjacent link button copies the canonical mneme://
// reference. `compact` hides the id text (for dense block/task controls),
// leaving only the reference button.
const props = withDefaults(
  defineProps<{ publicId: string; kind: Kind; docId?: string; compact?: boolean }>(),
  { compact: false },
)

const { toast } = useToast()
const reference = computed(() => formatRef(props.kind, props.publicId, props.docId))

async function copyId() {
  toast((await copyText(props.publicId)) ? `Copied ${props.publicId}` : 'Copy failed')
}
async function copyReference() {
  toast((await copyText(reference.value)) ? 'Copied reference' : 'Copy failed')
}
</script>

<template>
  <span class="ref-chip" :class="{ compact }">
    <button
      v-if="!compact"
      class="id mn-mono-sm"
      data-test="copy-id"
      :title="`Copy id ${publicId}`"
      @click="copyId"
    >{{ publicId }}</button>
    <button
      class="ref"
      data-test="copy-ref"
      :title="`Copy reference ${reference}`"
      :aria-label="`Copy reference ${reference}`"
      @click="copyReference"
    >
      <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true" focusable="false">
        <path
          d="M6.6 9.4l2.8-2.8M6.2 5H4.6a2.4 2.4 0 000 4.8h1.6m3.6 1H11.4a2.4 2.4 0 000-4.8H9.8"
          fill="none"
          stroke="currentColor"
          stroke-width="1.4"
          stroke-linecap="round"
        />
      </svg>
    </button>
  </span>
</template>

<style scoped>
.ref-chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  vertical-align: middle;
}
.id,
.ref {
  border: 1px solid var(--border-soft);
  background: var(--bg-surface);
  color: var(--text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  line-height: 1;
  transition:
    color var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}
.id {
  padding: 2px 6px;
  letter-spacing: 0.02em;
}
.ref {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  width: 20px;
  height: 20px;
}
.id:hover,
.ref:hover {
  color: var(--accent);
  border-color: var(--accent-border);
  background: var(--bg-hover);
}
.id:focus-visible,
.ref:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
</style>
