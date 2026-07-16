<script setup lang="ts">
import { renderInline } from '@/lib/markdown'

defineProps<{ id?: string; title?: string; cols?: string[]; rows?: string[][] }>()
</script>

<template>
  <div>
    <h4 v-if="title" class="mn-label mb-2">{{ title }}</h4>
    <div class="wrap">
      <table class="mn-body-sm">
        <thead v-if="cols?.length">
          <tr>
            <th v-for="(c, i) in cols" :key="i" class="mn-label">{{ c }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, ri) in rows ?? []" :key="ri">
            <td v-for="(cell, ci) in row" :key="ci" class="mn-md" v-html="renderInline(cell)" />
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
/* Two-tone: a shaded header row anchors a lighter body. The light body also
   lets the table read when nested inside a surface-colored card.

   Tables carry more columns than prose has words, so they break out wider than
   the reading measure — centred on the column's axis and bounded by the doc's
   available width, so a dense table breathes without ever pushing the page into
   a horizontal scrollbar. On narrow viewports min-width pins it to the column. */
.wrap {
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--bg-elevated);
  overflow-x: auto;
  width: min(
    1080px,
    calc(100vw - var(--rail-width) - var(--sidebar-width) - 2 * var(--space-8))
  );
  min-width: 100%;
  margin-left: 50%;
  transform: translateX(-50%);
}
table {
  width: 100%;
  border-collapse: collapse;
  font-variant-numeric: tabular-nums;
}
th {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-strong);
  vertical-align: bottom;
}
td {
  padding: var(--space-3) var(--space-4);
  vertical-align: top;
}
/* First column carries the row's subject — give it primary weight so wide,
   wrappy tables stay scannable down the left edge. */
td:first-child {
  color: var(--text-primary);
  font-weight: 500;
}
tbody tr + tr td {
  border-top: 1px solid var(--border);
}
</style>
