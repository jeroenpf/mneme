<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Document } from '@/types'

const props = defineProps<{ doc: Document }>()

const description = computed(() => {
  const d = props.doc.meta?.description
  return typeof d === 'string' ? d : ''
})

// phase_current is the phase in flight (1-based): phases below it are
// done, it is the active segment, the rest are todo.
const pips = computed(() => {
  const total = props.doc.phase_total ?? 0
  const current = props.doc.phase_current ?? 0
  return Array.from({ length: total }, (_, i) =>
    i + 1 < current ? 'done' : i + 1 === current ? 'wip' : 'todo',
  )
})

// Tags stay on one line; overflow is clipped. The full set surfaces via the
// native title tooltip, and a trailing fade only shows when tags are cut off.
const tagsTitle = computed(() => props.doc.tags.map((t) => `#${t}`).join(' '))

const tagsEl = ref<HTMLElement | null>(null)
const clipped = ref(false)

function measure() {
  const el = tagsEl.value
  if (!el) return
  clipped.value = el.scrollWidth > el.clientWidth + 1
}

let ro: ResizeObserver | undefined
onMounted(() => {
  measure()
  if (typeof ResizeObserver !== 'undefined' && tagsEl.value) {
    ro = new ResizeObserver(measure)
    ro.observe(tagsEl.value)
  }
})
onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <RouterLink :to="`/doc/${doc.id}`" class="doc-card">
    <div class="doc-card__head">
      <span class="status-dot" :class="`status-${doc.status}`" :title="doc.status" />
      <span class="mn-label">{{ doc.type }}</span>
      <div v-if="doc.ticket || doc.repo" class="doc-card__head-right">
        <span v-if="doc.ticket" class="mn-mono-sm">{{ doc.ticket }}</span>
        <span v-if="doc.repo" class="doc-card__area" data-test="repo" :title="doc.repo">
          ⎇ {{ doc.repo }}
        </span>
      </div>
    </div>

    <h3 class="mn-h2 line-clamp-2">{{ doc.title }}</h3>

    <p v-if="description" class="mn-body-sm line-clamp-2">{{ description }}</p>

    <div v-if="pips.length" class="flex items-center gap-1" data-test="pips">
      <span v-for="(p, i) in pips" :key="i" class="pip" :class="`pip-${p}`" />
      <span class="mn-mono-sm ml-1">{{ doc.phase_current }}/{{ doc.phase_total }}</span>
    </div>

    <div
      v-if="doc.tags.length"
      ref="tagsEl"
      class="doc-card__tags"
      :class="{ 'is-clipped': clipped }"
      :title="tagsTitle"
    >
      <span v-for="tag in doc.tags" :key="tag" class="doc-card__tag">#{{ tag }}</span>
    </div>
  </RouterLink>
</template>

<style scoped>
.doc-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  text-decoration: none;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}
.doc-card:hover {
  background: var(--bg-elevated);
  border-color: var(--border-strong);
}
.doc-card:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}

.doc-card__head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.doc-card__head-right {
  margin-left: auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.doc-card__head-right .mn-mono-sm {
  flex: none;
}
.doc-card__area {
  flex: 0 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  line-height: var(--lh-mono-sm);
  color: var(--text-muted);
}

.status-dot {
  width: 7px;
  height: 7px;
  flex: none;
  border-radius: var(--radius-xs);
}
.status-todo        { background: var(--status-todo); }
.status-in-progress { background: var(--status-wip); }
.status-complete    { background: var(--status-done); }
.status-blocked     { background: var(--status-blocked); }
.status-archived    { background: var(--status-archived); }

.pip {
  width: 14px;
  height: 4px;
  border-radius: 1px;
  background: var(--bg-overlay);
}
.pip-done { background: var(--accent); }
.pip-wip  { background: var(--accent-dim); box-shadow: inset 0 0 0 1px var(--accent-border); }

.doc-card__tags {
  margin-top: auto;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--space-2);
  min-width: 0;
  overflow: hidden;
}
.doc-card__tags.is-clipped {
  -webkit-mask-image: linear-gradient(to right, #000 calc(100% - 28px), transparent 100%);
  mask-image: linear-gradient(to right, #000 calc(100% - 28px), transparent 100%);
}
.doc-card__tag {
  flex: none;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  line-height: var(--lh-mono-sm);
  color: var(--text-muted);
}
</style>
