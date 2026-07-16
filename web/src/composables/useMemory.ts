import { ref, type Ref } from 'vue'
import { listMemory, type MemoryEntry } from '@/api/memory'
import type { RefreshOptions } from './refresh'

export interface AreaGroup {
  area: string
  entries: MemoryEntry[]
}

export interface ProjectGroup {
  project: string
  entries: MemoryEntry[] // project-scoped entries (no area)
  areas: AreaGroup[]
}

export interface GroupedMemory {
  global: MemoryEntry[]
  projects: ProjectGroup[]
}

const byKey = (a: MemoryEntry, b: MemoryEntry) => a.key.localeCompare(b.key)

// groupMemory folds a flat entry list into the scope hierarchy the page
// renders: a Global bucket, then one block per project (its own
// project-scoped entries plus nested area subsections). Everything is
// sorted stably by name/key so the view doesn't reshuffle on refresh.
export function groupMemory(entries: MemoryEntry[]): GroupedMemory {
  const global = entries.filter((e) => e.scope === 'global').sort(byKey)

  const projects = new Map<string, ProjectGroup>()
  const project = (slug: string): ProjectGroup => {
    let g = projects.get(slug)
    if (!g) {
      g = { project: slug, entries: [], areas: [] }
      projects.set(slug, g)
    }
    return g
  }

  for (const e of entries) {
    if (e.scope === 'project' && e.project) {
      project(e.project).entries.push(e)
    } else if (e.scope === 'area' && e.project && e.area) {
      const g = project(e.project)
      let area = g.areas.find((a) => a.area === e.area)
      if (!area) {
        area = { area: e.area, entries: [] }
        g.areas.push(area)
      }
      area.entries.push(e)
    }
  }

  const grouped = [...projects.values()].sort((a, b) => a.project.localeCompare(b.project))
  for (const g of grouped) {
    g.entries.sort(byKey)
    g.areas.sort((a, b) => a.area.localeCompare(b.area))
    for (const a of g.areas) a.entries.sort(byKey)
  }

  return { global, projects: grouped }
}

export interface UseMemoryResult {
  items: Ref<MemoryEntry[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useMemory(): UseMemoryResult {
  const items = ref<MemoryEntry[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      items.value = await listMemory()
      if (silent) error.value = null // recovered — show fresh content
    } catch (err) {
      if (silent) return // best-effort: keep the current list visible
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      if (!silent) loading.value = false
    }
  }

  void refresh()

  return { items, loading, error, refresh }
}
