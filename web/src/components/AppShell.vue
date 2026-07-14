<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import ThemePicker from './ThemePicker.vue'

// The persistent primary navigation — identical on every route. Order matches
// the mockup (design/theme-explorations.html): Registry first, then the stores.
const NAV = [
  { to: '/', label: 'Registry', test: 'to-registry' },
  { to: '/memory', label: 'Memory', test: 'to-memory' },
  { to: '/decisions', label: 'Decisions', test: 'to-decisions' },
  { to: '/snippets', label: 'Snippets', test: 'to-snippets' },
  { to: '/journal', label: 'Journal', test: 'to-journal' },
  { to: '/solutions', label: 'Solutions', test: 'to-solutions' },
  { to: '/env', label: 'Env', test: 'to-env' },
  { to: '/bundle', label: 'Bundle', test: 'to-bundle' },
] as const

// Global search (lifted from Topbar). A local ref — deliberately NOT wired to
// registry filtering; Enter routes to the global /search page for the query.
const search = ref('')
const input = ref<HTMLInputElement | null>(null)
const router = useRouter()

function onEnter() {
  const q = search.value.trim()
  if (q) router.push({ path: '/search', query: { q } })
}

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
  <div class="shell">
    <aside class="rail">
      <RouterLink to="/" class="brand">
        <span class="glyph">⬡</span><span class="word">mneme</span>
      </RouterLink>

      <div class="project"><b>mneme</b></div>

      <input
        ref="input"
        v-model="search"
        class="search mn-mono-sm"
        type="search"
        placeholder="search…  /"
        aria-label="Search documents"
        @keyup.enter="onEnter"
      />

      <nav class="nav" aria-label="Primary">
        <p class="nav-title mn-label">Knowledge</p>
        <RouterLink
          v-for="item in NAV"
          :key="item.to"
          :to="item.to"
          :data-test="item.test"
          class="nav-link"
        >
          <span class="mk" />
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="rail-foot">
        <p class="nav-title mn-label">Theme</p>
        <ThemePicker />
      </div>
    </aside>

    <main class="content"><slot /></main>
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: var(--rail-width) minmax(0, 1fr);
  min-height: 100vh;
}
.rail {
  position: sticky;
  top: 0;
  align-self: start;
  height: 100vh;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding: var(--space-5) var(--space-4);
  background: var(--bg-surface);
  border-right: 1px solid var(--border);
}
.brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px var(--space-2);
  text-decoration: none;
}
.glyph {
  color: var(--accent);
  font-size: 17px;
  line-height: 1;
}
.word {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 17px;
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}
.project {
  margin: 0 var(--space-1);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg);
  font-size: var(--fs-body-sm);
  color: var(--text-secondary);
}
.project b {
  color: var(--text-primary);
  font-weight: 600;
}
.search {
  width: 100%;
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 7px var(--space-3);
}
.search::placeholder {
  color: var(--text-faint);
}
.search:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.nav {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.nav-title {
  padding: 0 var(--space-3) var(--space-2);
}
.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 7px var(--space-3);
  border-radius: var(--radius);
  text-decoration: none;
  color: var(--text-secondary);
  font-size: var(--fs-body-sm);
  position: relative;
  transition:
    background var(--duration-fast),
    color var(--duration-fast);
}
.mk {
  width: 6px;
  height: 6px;
  border-radius: 2px;
  background: var(--text-faint);
  flex: none;
  transition: background var(--duration-fast);
}
.nav-link:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}
.nav-link:hover .mk {
  background: var(--text-muted);
}
.nav-link.router-link-exact-active {
  background: var(--accent-dim);
  color: var(--text-primary);
  font-weight: 500;
}
.nav-link.router-link-exact-active .mk {
  background: var(--accent);
}
.nav-link.router-link-exact-active::before {
  content: '';
  position: absolute;
  left: calc(-1 * var(--space-4));
  top: 6px;
  bottom: 6px;
  width: 3px;
  background: var(--accent);
  border-radius: 0 2px 2px 0;
}
.rail-foot {
  margin-top: auto;
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}
.rail-foot .nav-title {
  padding: 0 0 var(--space-2);
}
.content {
  min-width: 0;
}
</style>
