<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useEventStream } from '@/composables/useEventStream'
import { tryOpenRef } from '@/lib/openRef'
import ThemePicker from './ThemePicker.vue'
import ToastHost from './ToastHost.vue'

// Touching the shared stream here (the app shell mounts for the whole app's
// life) opens the singleton EventSource at boot, so live updates and the
// status dot work on every route — not only after a live view has mounted.
const { status } = useEventStream()
const statusLabel = computed(() =>
  status.value === 'open' ? 'connected' : status.value === 'connecting' ? 'connecting…' : 'disconnected',
)

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

// Operational surfaces — index health and diagnostics, distinct from the
// knowledge stores above.
const SYSTEM_NAV = [
  { to: '/embeddings', label: 'Search index', test: 'to-embeddings' },
  { to: '/help', label: 'Help', test: 'to-help' },
  { to: '/about', label: 'About', test: 'to-about' },
] as const

// Global search (lifted from Topbar). A local ref — deliberately NOT wired to
// registry filtering; Enter routes to the global /search page for the query.
const search = ref('')
const input = ref<HTMLInputElement | null>(null)
const router = useRouter()

function onEnter() {
  const q = search.value.trim()
  if (!q) return
  // A pasted mneme:// reference (or bare public id) jumps straight to its
  // target; anything else runs a full-text search.
  if (tryOpenRef(router, q)) {
    search.value = ''
    return
  }
  router.push({ path: '/search', query: { q } })
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
      <div class="brandbar">
        <RouterLink to="/" class="brand">
          <img class="mark" src="/mneme-mark.svg" alt="" aria-hidden="true" />
          <span class="word">mneme</span>
        </RouterLink>
        <span
          class="conn-dot"
          :class="`conn-${status}`"
          data-test="conn-status"
          :data-status="status"
          role="status"
          :aria-label="`Live updates: ${statusLabel}`"
          :title="`Live updates: ${statusLabel}`"
        />
      </div>

      <RouterLink to="/project/mneme" class="project" data-test="to-project-home">
        <b>mneme</b>
        <span class="project-hint mn-mono-sm">home</span>
      </RouterLink>

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

        <p class="nav-title nav-title-sys mn-label">System</p>
        <RouterLink
          v-for="item in SYSTEM_NAV"
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

    <ToastHost />
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
.brandbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px var(--space-2);
  text-decoration: none;
}
/* Live-connection indicator: green when streaming, amber (pulsing) while
   (re)connecting, red if the stream is closed. Pushed to the rail's edge. */
.conn-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
  margin-left: auto;
  margin-right: var(--space-2);
  background: var(--text-faint);
}
.conn-open {
  background: var(--green);
}
.conn-connecting {
  background: var(--yellow);
  animation: conn-pulse 1.2s ease-in-out infinite;
}
.conn-closed {
  background: var(--red);
}
@keyframes conn-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}
@media (prefers-reduced-motion: reduce) {
  .conn-connecting {
    animation: none;
  }
}
.mark {
  display: block;
  width: 21px;
  height: 21px;
  flex: none;
}
.word {
  font-family: var(--font-display);
  font-weight: 800;
  font-size: 17px;
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}
.project {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-2);
  margin: 0 var(--space-1);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg);
  font-size: var(--fs-body-sm);
  color: var(--text-secondary);
  text-decoration: none;
  transition:
    border-color var(--duration-fast),
    background var(--duration-fast);
}
.project:hover {
  border-color: var(--accent-border);
  background: var(--bg-hover);
}
.project:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.project b {
  color: var(--text-primary);
  font-weight: 600;
}
.project-hint {
  color: var(--text-faint);
}
.project.router-link-exact-active .project-hint {
  color: var(--accent);
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
.nav-title-sys {
  padding-top: var(--space-4);
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
