<script setup lang="ts">
import type { Solution } from '@/api/solutions'

defineProps<{ solution: Solution }>()

const fmtDate = (iso: string) => iso.slice(0, 10)
</script>

<template>
  <article class="solution" :data-test="`solution-${solution.id}`">
    <header class="head">
      <span class="err-label mn-label">error</span>
      <time class="date mn-mono-sm" data-test="date">{{ fmtDate(solution.created_at) }}</time>
    </header>
    <p class="error mn-body" data-test="error">{{ solution.error_description }}</p>

    <div class="fix">
      <span class="fix-label mn-label">fix</span>
      <p class="solution-text mn-body-sm" data-test="solution">{{ solution.solution }}</p>
    </div>

    <footer v-if="solution.tags.length || solution.source_url" class="meta">
      <span v-for="tag in solution.tags" :key="tag" class="tag mn-mono-sm" data-test="tag">#{{ tag }}</span>
      <a
        v-if="solution.source_url"
        :href="solution.source_url"
        class="source mn-mono-sm"
        target="_blank"
        rel="noopener noreferrer"
        data-test="source-url"
        >source ↗</a
      >
    </footer>
  </article>
</template>

<style scoped>
.solution {
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
.err-label {
  color: var(--red-text, var(--accent));
}
.date {
  color: var(--text-faint);
}
.error {
  color: var(--text-primary);
  margin: 0;
  overflow-wrap: anywhere;
}
.fix {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  border-left: 2px solid var(--border-strong);
  padding-left: var(--space-3);
}
.fix-label {
  color: var(--text-faint);
}
.solution-text {
  color: var(--text-secondary);
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.tag {
  color: var(--text-muted);
}
.source {
  color: var(--accent);
  text-decoration: none;
}
.source:hover {
  text-decoration: underline;
}
</style>
