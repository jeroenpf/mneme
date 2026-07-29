<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { getRelated, type RelatedBundle, type RelatedEntry } from '@/api/relations'
import { routeForRef } from '@/lib/mnemeRef'

// The relations view for one document: explicit typed links (both
// directions), the mentioned-by backlinks, and outgoing mentions. Renders
// nothing at all when the document has no relations — the panel is an
// enhancement, so a fetch failure also just hides it.
const props = defineProps<{ docId: string; revision?: number }>()

const bundle = ref<RelatedBundle | null>(null)

async function load() {
  try {
    bundle.value = await getRelated(props.docId)
  } catch {
    bundle.value = null
  }
}
// revision ties the panel to the document's SSE-driven reloads: a live
// update bumps the revision, which refreshes the edges too.
watch(() => [props.docId, props.revision], load, { immediate: true })

const isEmpty = computed(
  () =>
    !bundle.value ||
    (bundle.value.links.length === 0 &&
      bundle.value.mentions.length === 0 &&
      bundle.value.mentioned_by.length === 0),
)

// Directional labels: an incoming "implements" edge reads "implemented by".
// relates-to is symmetric and reads the same both ways.
const INCOMING: Record<string, string> = {
  implements: 'implemented by',
  supersedes: 'superseded by',
  'depends-on': 'depended on by',
  blocks: 'blocked by',
}

function relLabel(e: RelatedEntry): string {
  if (e.rel_type === 'relates-to') return 'relates to'
  if (e.direction === 'in') return INCOMING[e.rel_type] ?? `${e.rel_type} by`
  return e.rel_type.replace('-', ' ')
}

function routeFor(e: RelatedEntry) {
  return routeForRef({ kind: e.kind, id: e.id })
}
</script>

<template>
  <section v-if="!isEmpty && bundle" class="related" data-test="related-panel">
    <h2 class="mn-label">Related</h2>

    <ul v-if="bundle.links.length" class="group" data-test="related-links">
      <li v-for="e in bundle.links" :key="`${e.rel_type}:${e.direction}:${e.id}`">
        <span class="rel mn-mono-sm">{{ relLabel(e) }}</span>
        <RouterLink class="entry" :to="routeFor(e)">{{ e.title }}</RouterLink>
        <span v-if="e.kind !== 'document'" class="kind mn-mono-sm">{{ e.kind }}</span>
      </li>
    </ul>

    <template v-if="bundle.mentioned_by.length">
      <h3 class="mn-label">Mentioned by</h3>
      <ul class="group" data-test="related-backlinks">
        <li v-for="e in bundle.mentioned_by" :key="e.id">
          <RouterLink class="entry" :to="routeFor(e)">{{ e.title }}</RouterLink>
          <span v-if="e.kind !== 'document'" class="kind mn-mono-sm">{{ e.kind }}</span>
        </li>
      </ul>
    </template>

    <template v-if="bundle.mentions.length">
      <h3 class="mn-label">Mentions</h3>
      <ul class="group" data-test="related-mentions">
        <li v-for="e in bundle.mentions" :key="e.id">
          <span v-if="e.dangling" class="dangling mn-mono-sm">{{ e.title }}</span>
          <RouterLink v-else class="entry" :to="routeFor(e)">{{ e.title }}</RouterLink>
          <span v-if="e.kind !== 'document'" class="kind mn-mono-sm">{{ e.kind }}</span>
        </li>
      </ul>
    </template>
  </section>
</template>

<style scoped>
.related {
  margin-top: var(--space-8);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-subtle);
}
h3 {
  margin-top: var(--space-4);
}
.group {
  list-style: none;
  margin: var(--space-2) 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.rel {
  color: var(--text-muted);
  margin-right: var(--space-2);
}
.entry {
  color: var(--text);
  text-decoration: none;
}
.entry:hover {
  text-decoration: underline;
}
.kind {
  color: var(--text-faint);
  margin-left: var(--space-2);
}
.dangling {
  color: var(--text-faint);
}
</style>
