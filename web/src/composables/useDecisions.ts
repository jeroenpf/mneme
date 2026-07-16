import { ref, type Ref } from 'vue'
import { listDecisions, type Decision } from '@/api/decisions'
import type { RefreshOptions } from './refresh'

export interface ProjectDecisions {
  project: string // '' for global
  decisions: Decision[]
}

const newestFirst = (a: Decision, b: Decision) => b.created_at.localeCompare(a.created_at)

// groupDecisions buckets a flat list by project (global = ''), each bucket
// sorted newest-first. Buckets are ordered global-first, then project slug.
export function groupDecisions(decisions: Decision[]): ProjectDecisions[] {
  const byProject = new Map<string, Decision[]>()
  for (const d of decisions) {
    const key = d.project ?? ''
    const arr = byProject.get(key) ?? []
    arr.push(d)
    byProject.set(key, arr)
  }
  const groups: ProjectDecisions[] = [...byProject.entries()].map(([project, ds]) => ({
    project,
    decisions: [...ds].sort(newestFirst),
  }))
  groups.sort((a, b) => {
    if (a.project === '') return -1
    if (b.project === '') return 1
    return a.project.localeCompare(b.project)
  })
  return groups
}

export interface UseDecisionsResult {
  items: Ref<Decision[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useDecisions(): UseDecisionsResult {
  const items = ref<Decision[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      items.value = await listDecisions()
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
