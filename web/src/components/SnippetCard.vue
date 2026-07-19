<script setup lang="ts">
import type { Snippet } from '@/api/snippets'
import MCode from '@/blocks/MCode.vue'
import RefChip from './RefChip.vue'

defineProps<{ snippet: Snippet }>()
</script>

<template>
  <article class="snippet" :data-test="`snippet-${snippet.id}`">
    <header class="head">
      <h3 class="title mn-body">{{ snippet.title }}</h3>
      <div class="meta">
        <span v-if="snippet.language" class="lang mn-label" data-test="lang">{{ snippet.language }}</span>
        <span v-for="tag in snippet.tags" :key="tag" class="tag mn-mono-sm" data-test="tag">#{{ tag }}</span>
        <RefChip v-if="snippet.public_id" :public-id="snippet.public_id" kind="snippet" />
      </div>
    </header>
    <p v-if="snippet.description" class="desc mn-body-sm">{{ snippet.description }}</p>
    <MCode :lang="snippet.language || undefined" :content="snippet.content" />
  </article>
</template>

<style scoped>
.snippet {
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
  flex-wrap: wrap;
}
.title {
  color: var(--text-primary);
  overflow-wrap: anywhere;
}
.meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.lang {
  color: var(--text-faint);
}
.tag {
  color: var(--text-muted);
}
.desc {
  color: var(--text-secondary);
  margin: 0;
}
</style>
