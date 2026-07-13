<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { deleteEnv, setEnv, type EnvEntry } from '@/api/env'
import { useEnv } from '@/composables/useEnv'
import { useProjects } from '@/composables/useProjects'

const route = useRoute()
const router = useRouter()
const { items: projects, loading: projectsLoading } = useProjects()

// Selected project: seed from ?project=, else the first project once loaded.
const selected = ref<string>(typeof route.query.project === 'string' ? route.query.project : '')
watch(
  projects,
  (list) => {
    if (!selected.value && list.length) selected.value = list[0].slug
  },
  { immediate: true },
)
// Keep the URL shareable / deep-linkable.
watch(selected, (p) => {
  void router.replace({ query: p ? { project: p } : {} })
})

const { items, loading, error, refresh } = useEnv(selected)

const actionError = ref<string | null>(null)
async function run(fn: () => Promise<unknown>) {
  actionError.value = null
  try {
    await fn()
    await refresh()
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : String(err)
  }
}

function onValueBlur(entry: EnvEntry, ev: Event) {
  const next = (ev.target as HTMLInputElement).value.trim()
  if (!next || next === entry.value) return
  void run(() =>
    setEnv({ project: entry.project, key: entry.key, value: next, description: entry.description }),
  )
}

function onDescriptionBlur(entry: EnvEntry, ev: Event) {
  const next = (ev.target as HTMLInputElement).value.trim()
  if (next === (entry.description ?? '')) return
  void run(() =>
    setEnv({
      project: entry.project,
      key: entry.key,
      value: entry.value,
      description: next || undefined,
    }),
  )
}

function remove(entry: EnvEntry) {
  void run(() => deleteEnv({ project: entry.project, key: entry.key }))
}

const draft = reactive({ key: '', value: '', description: '' })
function add() {
  const key = draft.key.trim()
  const value = draft.value.trim()
  if (!key || !value || !selected.value) return
  void run(async () => {
    await setEnv({
      project: selected.value,
      key,
      value,
      description: draft.description.trim() || undefined,
    })
    draft.key = ''
    draft.value = ''
    draft.description = ''
  })
}

const isEmpty = computed(() => items.value.length === 0)
</script>

<template>
  <div>
    <header class="topbar">
      <RouterLink to="/" class="brand"><span class="glyph">⬡</span> mneme</RouterLink>
      <span class="mn-label">/ env</span>
      <RouterLink to="/" class="nav-link mn-mono-sm" data-test="to-registry">registry →</RouterLink>
    </header>

    <main class="content">
      <p class="mn-body-sm intro">
        Non-secret per-project config — ports, service names, local URLs, Docker service names.
        Claude Code reads it at session start via
        <span class="mn-code-inline">get_context_bundle</span> /
        <span class="mn-code-inline">get_env</span>.
      </p>

      <div class="secrets-warning" data-test="secrets-warning" role="alert">
        <span class="warn-icon">⚠</span>
        <span>
          <strong>Never store secrets here.</strong> No API keys, tokens, passwords, or
          credentialed connection strings — this registry is plaintext and unencrypted, for
          non-secret local config only.
        </span>
      </div>

      <div class="project-row">
        <label class="mn-mono-sm project-label" for="env-project">project</label>
        <select
          id="env-project"
          v-model="selected"
          class="project-select mn-mono"
          data-test="project-select"
        >
          <option v-for="p in projects" :key="p.slug" :value="p.slug">{{ p.slug }}</option>
        </select>
      </div>

      <p v-if="actionError" class="action-error mn-mono-sm" data-test="action-error">
        {{ actionError }}
      </p>

      <p v-if="projectsLoading && !projects.length" class="mn-mono-sm py-8 text-center">loading…</p>

      <p v-else-if="!projects.length" class="mn-body-sm empty" data-test="no-projects">
        no projects yet — create one over MCP with
        <span class="mn-code-inline">create_project</span> first.
      </p>

      <template v-else>
        <p v-if="loading" class="mn-mono-sm py-8 text-center">loading env…</p>

        <div v-else-if="error" class="error mn-body-sm" data-test="error">
          <p>could not load env: {{ error.message }}</p>
          <button class="retry mn-mono-sm" @click="refresh">retry</button>
        </div>

        <template v-else>
          <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
            no env entries for <strong>{{ selected }}</strong> yet — add one below, or record one
            over MCP with <span class="mn-code-inline">set_env</span>.
          </p>

          <div v-else class="rows" data-test="rows">
            <div class="row row-head mn-mono-sm">
              <span>key</span><span>value</span><span>description</span><span></span>
            </div>
            <div v-for="entry in items" :key="entry.id" class="row" data-test="entry">
              <span class="entry-key mn-mono">{{ entry.key }}</span>
              <input
                class="entry-value mn-mono"
                :value="entry.value"
                spellcheck="false"
                aria-label="env value"
                @blur="onValueBlur(entry, $event)"
                @keyup.enter="($event.target as HTMLInputElement).blur()"
              />
              <input
                class="entry-desc mn-mono"
                :value="entry.description ?? ''"
                spellcheck="false"
                placeholder="—"
                aria-label="env description"
                @blur="onDescriptionBlur(entry, $event)"
                @keyup.enter="($event.target as HTMLInputElement).blur()"
              />
              <button
                class="entry-delete"
                data-test="entry-delete"
                :aria-label="`delete ${entry.key}`"
                title="delete"
                @click="remove(entry)"
              >
                ×
              </button>
            </div>
          </div>

          <div class="add-row" data-test="add-row">
            <input
              v-model="draft.key"
              class="add-key mn-mono"
              placeholder="KEY"
              spellcheck="false"
              aria-label="new key"
              @keyup.enter="add"
            />
            <input
              v-model="draft.value"
              class="add-value mn-mono"
              placeholder="value"
              spellcheck="false"
              aria-label="new value"
              @keyup.enter="add"
            />
            <input
              v-model="draft.description"
              class="add-desc mn-mono"
              placeholder="description (optional)"
              spellcheck="false"
              aria-label="new description"
              @keyup.enter="add"
            />
            <button class="add-submit" data-test="add-submit" type="button" @click="add">add</button>
          </div>
        </template>
      </template>
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
  gap: var(--space-5);
  min-width: 0;
}
.intro {
  color: var(--text-muted);
  margin-top: calc(-1 * var(--space-2));
}

.secrets-warning {
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  color: var(--text-secondary);
}
.warn-icon {
  color: var(--red);
  font-size: 16px;
  line-height: 1.4;
}
.secrets-warning strong {
  color: var(--red);
}

.project-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.project-label {
  color: var(--text-faint);
}
.project-select {
  color: var(--text-primary);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.project-select:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
  border-color: transparent;
}

.action-error {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  color: var(--red);
}
.error {
  border: 1px solid var(--red-border);
  background: var(--red-dim);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  color: var(--text-secondary);
}
.retry {
  margin-top: var(--space-2);
  padding: 4px 10px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  cursor: pointer;
}
.empty {
  color: var(--text-muted);
}

.rows {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.row {
  display: grid;
  grid-template-columns: minmax(120px, 220px) minmax(0, 1fr) minmax(0, 1.4fr) auto;
  align-items: center;
  gap: var(--space-3);
}
.row-head {
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding-bottom: var(--space-1);
}
.entry-key {
  color: var(--accent);
  overflow-wrap: anywhere;
}
.entry-value,
.entry-desc,
.add-key,
.add-value,
.add-desc {
  color: var(--text-primary);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.entry-value:focus,
.entry-desc:focus,
.add-key:focus,
.add-value:focus,
.add-desc:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
  border-color: transparent;
}
.entry-desc::placeholder,
.add-key::placeholder,
.add-value::placeholder,
.add-desc::placeholder {
  color: var(--text-faint);
}
.entry-delete {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-faint);
  font-size: 16px;
  line-height: 1;
  cursor: pointer;
}
.entry-delete:hover {
  color: var(--red);
  border-color: var(--red-border);
  background: var(--red-dim);
}

.add-row {
  display: grid;
  grid-template-columns: minmax(120px, 220px) minmax(0, 1fr) minmax(0, 1.4fr) auto;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-2);
}
.add-submit {
  padding: 5px 12px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: var(--fs-mono-sm);
  cursor: pointer;
}
.add-submit:hover {
  color: var(--accent);
  border-color: var(--accent-border);
}
</style>
