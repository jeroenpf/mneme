<script setup lang="ts">
defineProps<{ id?: string; headers?: string[]; rows?: string[][] }>()
</script>

<template>
  <table class="mn-body-sm border border-border rounded-md">
    <thead v-if="headers">
      <tr>
        <th v-for="(h, i) in headers" :key="i" class="text-left p-2 mn-label">{{ h }}</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(row, ri) in rows ?? []" :key="ri">
        <td v-for="(cell, ci) in row" :key="ci" class="p-2">{{ cell }}</td>
      </tr>
    </tbody>
  </table>
</template>
