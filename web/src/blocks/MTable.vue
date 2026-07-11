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
.wrap {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
}
th {
  text-align: left;
  padding: var(--space-2) var(--space-4);
  border-bottom: 1px solid var(--border);
}
td {
  padding: var(--space-2) var(--space-4);
}
tbody tr + tr td {
  border-top: 1px solid var(--border-soft);
}
</style>
