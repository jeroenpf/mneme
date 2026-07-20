<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { reindexFailed, searchStatus, type SearchStatus } from '@/api/search'
import { useToast } from '@/composables/useToast'

// Operational view of the embedding index: provider status, live queue depth,
// last reconciliation, per-type coverage buckets, and a manual retry for
// terminally-failed sources. Read-only apart from the retry action.
const status = ref<SearchStatus | null>(null)
const loading = ref(true)
const error = ref<Error | null>(null)
const retrying = ref(false)
const { toast } = useToast()

async function refresh() {
  try {
    status.value = await searchStatus()
    error.value = null
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
  } finally {
    loading.value = false
  }
}
onMounted(refresh)

const items = computed(() => status.value?.items ?? [])
const totalFailed = computed(() => items.value.reduce((n, i) => n + i.failed, 0))
const totalEmbedded = computed(() => items.value.reduce((n, i) => n + i.embedded, 0))
const totalSources = computed(() => items.value.reduce((n, i) => n + i.total, 0))

async function retry() {
  retrying.value = true
  try {
    const { retried } = await reindexFailed()
    toast(retried > 0 ? `Re-queued ${retried} failed source${retried === 1 ? '' : 's'}` : 'Nothing to retry')
    await refresh()
  } catch (err) {
    toast(err instanceof Error ? err.message : 'Retry failed')
  } finally {
    retrying.value = false
  }
}

// Relative time for the last reconciliation — it is a same-process signal, so a
// coarse "Nm ago" is more legible than an absolute timestamp.
function relTime(iso?: string): string {
  if (!iso) return 'never'
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
  <main class="page">
    <header class="head">
      <p class="mn-label eyebrow">embeddings</p>
      <h1 class="mn-h1">Search index</h1>
    </header>

    <p v-if="loading" class="mn-mono-sm state" data-test="loading">loading…</p>

    <div v-else-if="error" class="error-box mn-body-sm" data-test="error">
      could not load index status: {{ error.message }}
      <button class="ghost mn-mono-sm" @click="refresh">retry</button>
    </div>

    <template v-else-if="status">
      <!-- Provider identity + opt-in state. -->
      <div class="provider" data-test="provider">
        <template v-if="status.enabled">
          <span class="dot on" aria-hidden="true" />
          <span class="mn-body-sm">
            Semantic search via <b>{{ status.provider.name }}</b>
            <span class="mn-mono-sm model">{{ status.provider.model }}</span>
          </span>
        </template>
        <template v-else>
          <span class="dot off" aria-hidden="true" />
          <span class="mn-body-sm" data-test="disabled">
            Lexical-only search — no embedding provider configured.
          </span>
        </template>
      </div>

      <!-- Live operational counters. -->
      <div class="strip">
        <div class="cell">
          <div class="mn-label">coverage</div>
          <div class="num">{{ totalEmbedded }}<span class="of">/{{ totalSources }}</span></div>
        </div>
        <div class="cell" data-test="queue-depth">
          <div class="mn-label">queue</div>
          <div class="num" :class="{ active: status.queue_depth > 0 }">{{ status.queue_depth }}</div>
        </div>
        <div class="cell" data-test="failed-total">
          <div class="mn-label">failed</div>
          <div class="num" :class="{ bad: totalFailed > 0 }">{{ totalFailed }}</div>
        </div>
        <div class="cell" data-test="last-reconcile">
          <div class="mn-label">last reconcile</div>
          <div class="num sm">{{ relTime(status.last_reconcile) }}</div>
        </div>
      </div>

      <div v-if="status.enabled && totalFailed > 0" class="actions">
        <button class="retry mn-mono-sm" data-test="retry-failed" :disabled="retrying" @click="retry">
          {{ retrying ? 're-queuing…' : `retry ${totalFailed} failed` }}
        </button>
        <span class="mn-body-sm hint">re-enqueues failed sources for another embedding attempt</span>
      </div>

      <!-- Per-type coverage buckets. -->
      <div class="table-wrap">
        <table class="cov">
          <thead>
            <tr>
              <th class="mn-label">type</th>
              <th class="mn-label num-col">embedded</th>
              <th class="mn-label num-col">missing</th>
              <th class="mn-label num-col">stale</th>
              <th class="mn-label num-col">orphaned</th>
              <th class="mn-label num-col">failed</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in items" :key="it.type" data-test="type-row">
              <td class="type">{{ it.type }}</td>
              <td class="num-col mn-mono-sm">{{ it.embedded }}<span class="of">/{{ it.total }}</span></td>
              <td class="num-col mn-mono-sm" :class="{ warn: it.missing > 0 }">{{ it.missing }}</td>
              <td class="num-col mn-mono-sm" :class="{ warn: it.stale > 0 }">{{ it.stale }}</td>
              <td class="num-col mn-mono-sm" :class="{ warn: it.orphaned > 0 }">{{ it.orphaned }}</td>
              <td class="num-col mn-mono-sm" :class="{ bad: it.failed > 0 }">{{ it.failed }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </main>
</template>

<style scoped>
.page {
  max-width: 860px;
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}
.eyebrow {
  color: var(--eyebrow);
  margin-bottom: var(--space-1);
}
.state {
  color: var(--text-muted);
  padding: var(--space-8) 0;
  text-align: center;
}
.error-box {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.ghost {
  margin-left: var(--space-3);
  padding: 3px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}

.provider {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
}
.dot.on {
  background: var(--green);
}
.dot.off {
  background: var(--text-faint);
}
.model {
  color: var(--text-muted);
  margin-left: var(--space-2);
}

.strip {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
  min-width: 0;
}
.cell + .cell {
  border-left: 1px solid var(--border-soft);
}
.num {
  font-family: var(--font-mono);
  font-size: var(--fs-h1);
  line-height: var(--lh-h1);
  font-weight: 600;
  color: var(--text-primary);
}
.num.sm {
  font-size: var(--fs-h3);
  line-height: 1.2;
  color: var(--text-secondary);
}
.num.active {
  color: var(--status-wip);
}
.num.bad {
  color: var(--red);
}
.of {
  color: var(--text-faint);
  font-weight: 400;
}

.actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.retry {
  padding: 6px 14px;
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-sm);
  background: var(--accent-dim);
  color: var(--accent);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}
.retry:hover:not(:disabled) {
  background: var(--accent);
  color: var(--bg-elevated);
}
.retry:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.hint {
  color: var(--text-faint);
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
}
.cov {
  width: 100%;
  border-collapse: collapse;
}
.cov th,
.cov td {
  text-align: left;
  padding: var(--space-2) var(--space-4);
}
.cov thead th {
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.cov tbody tr + tr td {
  border-top: 1px solid var(--border-soft);
}
.cov tbody tr:nth-child(even) {
  background: var(--bg);
}
.type {
  color: var(--text-primary);
  font-weight: 500;
}
.num-col {
  text-align: right;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}
.num-col.warn {
  color: var(--yellow);
}
.num-col.bad {
  color: var(--red);
}
</style>
