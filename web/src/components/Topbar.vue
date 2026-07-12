<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const model = defineModel<string>({ default: '' })
const input = ref<HTMLInputElement | null>(null)

// Terminal affordance: `/` focuses search from anywhere on the page.
function onKeydown(e: KeyboardEvent) {
  if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)) return
  e.preventDefault()
  input.value?.focus()
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <header class="topbar">
    <span class="brand"><span class="glyph">⬡</span> mneme</span>
    <span class="mn-label">/ registry</span>
    <RouterLink to="/memory" class="nav-link mn-mono-sm" data-test="to-memory">memory →</RouterLink>
    <RouterLink to="/decisions" class="nav-link mn-mono-sm" data-test="to-decisions">decisions →</RouterLink>
    <input
      ref="input"
      v-model="model"
      class="search mn-mono-sm"
      type="search"
      placeholder="search…  /"
      aria-label="Search documents"
    />
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  height: var(--topbar-height);
  padding: 0 var(--space-6);
  border-bottom: 1px solid var(--border);
}
.brand {
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 16px;
  color: var(--text-primary);
}
.glyph {
  color: var(--accent);
}
.nav-link {
  color: var(--text-muted);
  text-decoration: none;
}
.nav-link:hover {
  color: var(--accent);
}
.search {
  margin-left: auto;
  width: 280px;
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 6px 10px;
}
.search::placeholder {
  color: var(--text-faint);
}
.search:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
}
</style>
