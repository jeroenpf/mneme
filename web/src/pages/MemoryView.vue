<script setup lang="ts">
import { computed, reactive, ref, watchEffect } from 'vue'
import {
  deleteMemory,
  setMemory,
  type MemoryEntry,
  type MemoryTarget,
} from '@/api/memory'
import { groupMemory, useMemory } from '@/composables/useMemory'
import { useLiveRefresh } from '@/composables/useLiveRefresh'

const { items, loading, error, refresh } = useMemory()
const grouped = computed(() => groupMemory(items.value))

// Live updates: when the agent writes memory over MCP, silently refetch
// (view stays mounted) and flash the changed entry. The event carries the
// key as its id; keys can repeat across scopes, so this flashes the first
// match — the live layer is convenience, correctness is the refetch.
useLiveRefresh('memory', {
  refresh: () => refresh({ silent: true }),
  flashTarget: (ev) => `[data-flash-id="${ev.id}"]`,
})

type AddTarget = Omit<MemoryTarget, 'key'>

interface Section {
  id: string
  title: string
  level: 'global' | 'project' | 'area'
  target: AddTarget
  entries: MemoryEntry[]
}

// Flatten the scope hierarchy into a single ordered list so the template
// renders every scope — Global, each project, each area — through one
// uniform block (rows + add form).
const sections = computed<Section[]>(() => {
  const g = grouped.value
  const out: Section[] = [
    { id: 'global', title: 'Global', level: 'global', target: { scope: 'global' }, entries: g.global },
  ]
  for (const p of g.projects) {
    out.push({
      id: `project:${p.project}`,
      title: p.project,
      level: 'project',
      target: { scope: 'project', project: p.project },
      entries: p.entries,
    })
    for (const a of p.areas) {
      out.push({
        id: `area:${p.project}:${a.area}`,
        title: a.area,
        level: 'area',
        target: { scope: 'area', project: p.project, area: a.area },
        entries: a.entries,
      })
    }
  }
  return out
})

const isEmpty = computed(() => items.value.length === 0)

// One draft (key + value) per add-form, keyed by section id. Initialized
// pre-render as sections appear so v-model always has a target.
const drafts = reactive<Record<string, { key: string; value: string }>>({})
watchEffect(() => {
  for (const s of sections.value) {
    if (!drafts[s.id]) drafts[s.id] = { key: '', value: '' }
  }
})

// Mutations share one status line; a failure (e.g. unknown project on a
// project-scoped add) surfaces there rather than vanishing.
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

function onBlur(entry: MemoryEntry, ev: Event) {
  const next = (ev.target as HTMLInputElement).value.trim()
  if (!next || next === entry.value) return
  void run(() =>
    setMemory({
      scope: entry.scope,
      key: entry.key,
      project: entry.project,
      area: entry.area,
      value: next,
    }),
  )
}

function remove(entry: MemoryEntry) {
  void run(() =>
    deleteMemory({ scope: entry.scope, key: entry.key, project: entry.project, area: entry.area }),
  )
}

function add(section: Section) {
  const d = drafts[section.id]
  const key = d.key.trim()
  const value = d.value.trim()
  if (!key || !value) return
  void run(async () => {
    await setMemory({ ...section.target, key, value })
    d.key = ''
    d.value = ''
  })
}
</script>

<template>
  <div>
    <main class="content">
      <p class="mn-body-sm intro">
        Persistent key/value context, layered global → project → area. Claude Code loads it at
        session start via <span class="mn-code-inline">get_memory</span>; edit or seed it here.
      </p>

      <p v-if="actionError" class="action-error mn-mono-sm" data-test="action-error">
        {{ actionError }}
      </p>

      <p v-if="loading" class="mn-mono-sm py-8 text-center">loading memory…</p>

      <div v-else-if="error" class="error mn-body-sm" data-test="error">
        <p>could not load memory: {{ error.message }}</p>
        <button class="retry mn-mono-sm" @click="refresh()">retry</button>
      </div>

      <template v-else>
        <p v-if="isEmpty" class="mn-body-sm empty" data-test="empty">
          no memory yet — add a global entry below, or record one over MCP with
          <span class="mn-code-inline">set_memory</span>.
        </p>

        <section
          v-for="section in sections"
          :key="section.id"
          class="scope"
          :class="`level-${section.level}`"
          data-test="scope"
        >
          <h2 class="scope-title" :class="section.level === 'global' ? 'mn-h2' : 'mn-h3'">
            <span v-if="section.level === 'area'" class="scope-prefix mn-mono-sm">area /</span>
            <span v-else-if="section.level === 'project'" class="scope-prefix mn-mono-sm">project /</span>
            {{ section.title }}
          </h2>

          <div v-if="section.entries.length" class="rows">
            <div
              v-for="entry in section.entries"
              :key="entry.id"
              class="entry"
              :data-flash-id="entry.key"
              data-test="entry"
            >
              <span class="entry-key mn-mono">{{ entry.key }}</span>
              <input
                class="entry-value mn-mono"
                :value="entry.value"
                spellcheck="false"
                aria-label="memory value"
                @blur="onBlur(entry, $event)"
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
          <p v-else class="scope-empty mn-mono-sm">— empty —</p>

          <div class="add-row" :data-test="`add-${section.id}`">
            <input
              v-model="drafts[section.id].key"
              class="add-key mn-mono"
              placeholder="key"
              spellcheck="false"
              aria-label="new key"
              @keyup.enter="add(section)"
            />
            <input
              v-model="drafts[section.id].value"
              class="add-value mn-mono"
              placeholder="value"
              spellcheck="false"
              aria-label="new value"
              @keyup.enter="add(section)"
            />
            <button
              class="add-submit"
              data-test="add-submit"
              type="button"
              @click="add(section)"
            >
              add
            </button>
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
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

.scope {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.scope.level-project,
.scope.level-area {
  border-left: 1px solid var(--border-soft);
  padding-left: var(--space-4);
}
.scope.level-area {
  margin-left: var(--space-4);
}
.scope-title {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  margin-bottom: var(--space-1);
}
.scope-prefix {
  color: var(--text-faint);
  text-transform: none;
}
.scope-empty {
  color: var(--text-faint);
  padding: var(--space-1) 0;
}

.rows {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.entry {
  display: grid;
  grid-template-columns: minmax(120px, 200px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
}
.entry-key {
  color: var(--accent);
  overflow-wrap: anywhere;
}
.entry-value,
.add-key,
.add-value {
  color: var(--text-primary);
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
}
.entry-value:focus,
.add-key:focus,
.add-value:focus {
  outline: none;
  box-shadow: var(--shadow-focus);
  border-color: transparent;
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
  grid-template-columns: minmax(120px, 200px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-1);
}
.add-key::placeholder,
.add-value::placeholder {
  color: var(--text-faint);
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
