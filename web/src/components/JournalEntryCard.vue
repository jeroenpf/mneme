<script setup lang="ts">
import type { JournalEntry } from '@/api/journal'

defineProps<{ entry: JournalEntry }>()

const fmtDate = (iso: string) => iso.slice(0, 10)
</script>

<template>
  <article class="entry" :data-test="`entry-${entry.id}`">
    <header class="head">
      <span v-if="entry.session_ref" class="session mn-mono-sm" data-test="session-ref">{{ entry.session_ref }}</span>
      <time class="date mn-mono-sm" data-test="date">{{ fmtDate(entry.created_at) }}</time>
    </header>
    <p class="summary mn-body">{{ entry.summary }}</p>

    <ul v-if="entry.accomplished.length" class="list done">
      <li v-for="(a, i) in entry.accomplished" :key="i" class="mn-body-sm" data-test="accomplished">
        <span class="marker">✓</span> {{ a }}
      </li>
    </ul>
    <ul v-if="entry.deferred.length" class="list later">
      <li v-for="(d, i) in entry.deferred" :key="i" class="mn-body-sm" data-test="deferred">
        <span class="marker">→</span> {{ d }}
      </li>
    </ul>
  </article>
</template>

<style scoped>
.entry {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
}
.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
}
.session {
  color: var(--accent);
}
.date {
  color: var(--text-faint);
}
.summary {
  color: var(--text-primary);
  margin: 0;
  overflow-wrap: anywhere;
}
.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.list.done {
  color: var(--text-secondary);
}
.list.later {
  color: var(--text-muted);
}
.marker {
  display: inline-block;
  width: 1.2em;
  color: var(--text-faint);
}
</style>
