<script setup lang="ts">
import { useTheme, type Theme } from '@/composables/useTheme'

const { current, THEMES, setTheme } = useTheme()

const label = (t: Theme) => t.charAt(0).toUpperCase() + t.slice(1)
const swatch = (t: Theme) => `var(--swatch-${t})`
</script>

<template>
  <div class="seg" role="group" aria-label="Theme">
    <button
      v-for="t in THEMES"
      :key="t"
      type="button"
      class="seg-btn mn-mono-sm"
      :data-test="`theme-${t}`"
      :aria-pressed="current === t ? 'true' : 'false'"
      @click="setTheme(t)"
    >
      <i class="dot" :style="{ background: swatch(t) }" />
      {{ label(t) }}
    </button>
  </div>
</template>

<style scoped>
.seg {
  display: inline-flex;
  padding: 4px;
  gap: 3px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.seg-btn {
  appearance: none;
  border: 0;
  cursor: pointer;
  color: var(--text-muted);
  background: transparent;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  gap: 7px;
  transition:
    color var(--duration-fast),
    background var(--duration-fast);
}
.seg-btn:hover {
  color: var(--text-primary);
}
.seg-btn[aria-pressed='true'] {
  color: var(--text-primary);
  background: var(--bg-elevated);
  font-weight: 600;
}
.seg-btn:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.dot {
  width: 11px;
  height: 11px;
  border-radius: var(--radius-pill);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}
</style>
