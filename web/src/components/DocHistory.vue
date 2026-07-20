<script setup lang="ts">
import { ref } from 'vue'
import { listRevisions, restoreRevision, type DocRevision } from '@/api/documents'
import { useToast } from '@/composables/useToast'

// The document's revision history with restore. Collapsed by default — it loads
// its list only when opened, so it never taxes a plain document read. Restoring
// a past revision writes a new forward revision (history is append-only); the
// parent refetches the document on the `restored` event.
const props = defineProps<{ docId: string; currentRevision?: number }>()
const emit = defineEmits<{ restored: [] }>()

const open = ref(false)
const revisions = ref<DocRevision[]>([])
const loading = ref(false)
const error = ref<Error | null>(null)
const restoringRev = ref<number | null>(null)
const loaded = ref(false)
const { toast } = useToast()

async function load() {
  loading.value = true
  error.value = null
  try {
    revisions.value = await listRevisions(props.docId)
    loaded.value = true
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
  } finally {
    loading.value = false
  }
}

function toggle() {
  open.value = !open.value
  if (open.value && !loaded.value) void load()
}

async function restore(rev: number) {
  restoringRev.value = rev
  try {
    const res = await restoreRevision(props.docId, rev)
    toast(`Restored revision ${rev} → new revision ${res.new_revision}`)
    emit('restored')
    await load()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Restore failed')
  } finally {
    restoringRev.value = null
  }
}

function relTime(iso: string): string {
  const secs = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (secs < 45) return 'just now'
  const mins = Math.round(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.round(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.round(hrs / 24)}d ago`
}
</script>

<template>
  <section class="history">
    <button
      class="toggle mn-label"
      data-test="history-toggle"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="chevron" :class="{ open }" aria-hidden="true">▸</span>
      history
      <span v-if="loaded" class="count mn-mono-sm">{{ revisions.length }}</span>
    </button>

    <div v-if="open" class="panel">
      <p v-if="loading" class="mn-mono-sm state">loading history…</p>
      <p v-else-if="error" class="mn-body-sm state err">could not load history: {{ error.message }}</p>
      <ol v-else class="rows">
        <li v-for="rev in revisions" :key="rev.revision" class="row" data-test="rev-row">
          <span class="rev mn-mono-sm">r{{ rev.revision }}</span>
          <span class="body">
            <span class="op mn-mono-sm">{{ rev.op }}</span>
            <span class="actor mn-mono-sm" :class="`actor-${rev.actor}`">{{ rev.actor }}</span>
            <span v-if="rev.target_ids?.length" class="targets mn-mono-sm">
              {{ rev.target_ids.length }} block{{ rev.target_ids.length === 1 ? '' : 's' }}
            </span>
          </span>
          <time class="when mn-mono-sm" :title="rev.created_at">{{ relTime(rev.created_at) }}</time>
          <button
            v-if="rev.revision !== currentRevision"
            class="restore mn-mono-sm"
            data-test="restore"
            :disabled="restoringRev !== null"
            @click="restore(rev.revision)"
          >
            {{ restoringRev === rev.revision ? '…' : 'restore' }}
          </button>
          <span v-else class="current mn-mono-sm">current</span>
        </li>
      </ol>
    </div>
  </section>
</template>

<style scoped>
.history {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  overflow: hidden;
}
.toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-3) var(--space-4);
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  text-align: left;
}
.toggle:hover {
  color: var(--text-secondary);
}
.chevron {
  display: inline-block;
  transition: transform var(--duration-fast) var(--ease-out);
  color: var(--text-faint);
}
.chevron.open {
  transform: rotate(90deg);
}
.count {
  color: var(--text-faint);
}
.panel {
  border-top: 1px solid var(--border-soft);
}
.state {
  padding: var(--space-3) var(--space-4);
  color: var(--text-muted);
}
.state.err {
  color: var(--red);
}
.rows {
  list-style: none;
  margin: 0;
  padding: 0;
}
.row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
}
.row + .row {
  border-top: 1px solid var(--border-soft);
}
.row:nth-child(even) {
  background: var(--bg);
}
.rev {
  color: var(--text-faint);
  flex: none;
  width: 2.5rem;
}
.body {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex: 1;
  min-width: 0;
  flex-wrap: wrap;
}
.op {
  color: var(--text-primary);
}
.actor {
  padding: 1px 6px;
  border-radius: var(--radius-pill);
  background: var(--bg-overlay);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-label);
  font-size: 9px;
}
.actor-mcp {
  color: var(--accent);
  background: var(--accent-dim);
}
.targets {
  color: var(--text-faint);
}
.when {
  color: var(--text-faint);
  flex: none;
}
.restore {
  flex: none;
  padding: 3px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  cursor: pointer;
  transition:
    color var(--duration-fast),
    border-color var(--duration-fast);
}
.restore:hover:not(:disabled) {
  color: var(--accent);
  border-color: var(--accent-border);
}
.restore:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.current {
  flex: none;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: var(--tracking-label);
  font-size: 9px;
}
</style>
