<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listProjects } from '@/api/projects'
import { getBundle, type Bundle } from '@/api/bundle'
import type { ProjectStats } from '@/types'

const projects = ref<ProjectStats[]>([])
const selected = ref('')
const area = ref('')
const bundle = ref<Bundle | null>(null)
const loading = ref(false)
const error = ref<Error | null>(null)
const copied = ref(false)

onMounted(async () => {
  try {
    projects.value = await listProjects()
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
  }
})

async function load() {
  if (!selected.value) {
    bundle.value = null
    return
  }
  loading.value = true
  error.value = null
  try {
    bundle.value = await getBundle(selected.value, area.value || undefined)
  } catch (err) {
    error.value = err instanceof Error ? err : new Error(String(err))
    bundle.value = null
  } finally {
    loading.value = false
  }
}

async function copyDigest() {
  if (!bundle.value) return
  await navigator.clipboard.writeText(bundle.value.markdown)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <div>
    <header class="topbar">
      <RouterLink to="/" class="brand"><span class="glyph">⬡</span> mneme</RouterLink>
      <span class="mn-label">/ bundle</span>
      <RouterLink to="/" class="nav-link mn-mono-sm" data-test="to-registry">registry →</RouterLink>
    </header>

    <main class="content">
      <p class="mn-body-sm intro">
        The context bundle — everything a Claude Code session needs for a project, assembled in one call
        (<span class="mn-code-inline">get_context_bundle</span>). Pick a project to preview what a session receives.
      </p>

      <div class="controls">
        <select
          v-model="selected"
          class="filter mn-mono-sm"
          aria-label="project"
          data-test="project-select"
          @change="load"
        >
          <option value="">select a project…</option>
          <option v-for="p in projects" :key="p.slug" :value="p.slug">{{ p.name }}</option>
        </select>
        <input
          v-model="area"
          class="filter mn-mono-sm"
          placeholder="area (optional)"
          aria-label="area"
          data-test="area-input"
          @keyup.enter="load"
        />
        <button class="load mn-mono-sm" data-test="load" :disabled="!selected" @click="load">load</button>
      </div>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">assembling…</p>
      <div v-else-if="error" class="error-box mn-body-sm" data-test="error">
        could not load bundle: {{ error.message }}
      </div>

      <template v-else-if="bundle">
        <div class="digest-head">
          <span class="mn-label">session digest</span>
          <button class="copy mn-mono-sm" data-test="copy" @click="copyDigest">
            {{ copied ? 'copied ✓' : 'copy' }}
          </button>
        </div>
        <pre class="digest mn-mono-sm" data-test="digest">{{ bundle.markdown }}</pre>
      </template>

      <p v-else class="mn-body-sm empty" data-test="empty">select a project to preview its context bundle.</p>
    </main>
  </div>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  height: var(--topbar-height);
  padding: 0 var(--space-6);
  background: var(--bg);
  border-bottom: 1px solid var(--border);
}
.brand {
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 16px;
  color: var(--text-primary);
  text-decoration: none;
}
.glyph {
  color: var(--accent);
}
.nav-link {
  margin-left: auto;
  color: var(--text-muted);
  text-decoration: none;
}
.nav-link:hover {
  color: var(--accent);
}

.content {
  max-width: var(--content-max);
  width: 100%;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  min-width: 0;
}
.intro {
  color: var(--text-muted);
  margin-top: calc(-1 * var(--space-2));
}
.controls {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.filter {
  color: var(--text-primary);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.filter:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
}
.load,
.copy {
  padding: 5px 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}
.load:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.error-box {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.empty {
  color: var(--text-muted);
}
.digest-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.digest {
  margin: 0;
  padding: var(--space-4);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  color: var(--text-primary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  overflow-x: auto;
}
</style>
