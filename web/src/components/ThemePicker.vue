<script setup lang="ts">
import { useTheme, type Theme } from '@/composables/useTheme'

const { current, THEMES, setTheme } = useTheme()

const label = (t: Theme) => t.charAt(0).toUpperCase() + t.slice(1)
const swatch = (t: Theme) => `var(--swatch-${t})`
</script>

<template>
  <div class="picker" role="group" aria-label="Theme">
    <div class="dots">
      <button
        v-for="t in THEMES"
        :key="t"
        type="button"
        class="dot-btn"
        :data-test="`theme-${t}`"
        :title="label(t)"
        :aria-label="label(t)"
        :aria-pressed="current === t ? 'true' : 'false'"
        @click="setTheme(t)"
      >
        <i class="dot" :style="{ background: swatch(t) }" />
      </button>
    </div>
    <span class="name mn-mono-sm" data-test="theme-name">{{ label(current) }}</span>
  </div>
</template>

<style scoped>
/* One swatch dot per theme, active name at the right — scales to any theme
   count inside the 224px rail: the dot row wraps, the name yields (ellipsis)
   rather than pushing the row past the rail's inner width. */
.picker {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  width: 100%;
  padding: 3px 8px 3px 3px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}
.dots {
  display: flex;
  flex-wrap: wrap;
  gap: 1px;
}
.dot-btn {
  appearance: none;
  border: 0;
  cursor: pointer;
  background: transparent;
  width: 24px;
  height: 24px;
  padding: 0;
  border-radius: var(--radius-pill);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.dot {
  width: 12px;
  height: 12px;
  border-radius: var(--radius-pill);
  box-shadow: inset 0 0 0 1px var(--border-strong);
  transition: box-shadow var(--duration-fast);
}
.dot-btn:hover .dot {
  box-shadow:
    inset 0 0 0 1px var(--border-strong),
    0 0 0 2px var(--bg-hover);
}
.dot-btn[aria-pressed='true'] .dot {
  box-shadow:
    inset 0 0 0 1px var(--border-strong),
    0 0 0 1.5px var(--bg-surface),
    0 0 0 3px var(--accent);
}
.dot-btn:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}
</style>
