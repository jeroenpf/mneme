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
/* Fills the rail's inner width so the three options never overflow the
   224px rail (they used to spill past its right edge at inline-flex's
   natural width). Buttons flex to share the width evenly. */
.seg {
  display: flex;
  width: 100%;
  padding: 3px;
  gap: 2px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.seg-btn {
  flex: 1 1 0;
  min-width: 0;
  appearance: none;
  border: 0;
  cursor: pointer;
  color: var(--text-muted);
  background: transparent;
  padding: 6px 2px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  white-space: nowrap;
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
  width: 9px;
  height: 9px;
  flex: none;
  border-radius: var(--radius-pill);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}
</style>
