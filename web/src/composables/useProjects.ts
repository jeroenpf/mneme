import { ref, type Ref } from 'vue'
import { listProjects } from '@/api/projects'
import type { ProjectStats } from '@/types'
import type { RefreshOptions } from './refresh'

export interface RegistryCounts {
  total: number
  inProgress: number
  complete: number
  todo: number
}

export interface UseProjectsResult {
  items: Ref<ProjectStats[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useProjects(): UseProjectsResult {
  const items = ref<ProjectStats[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      items.value = await listProjects()
      if (silent) error.value = null // recovered — show fresh content
    } catch (err) {
      if (silent) return // best-effort: keep the current stats visible
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      if (!silent) loading.value = false
    }
  }

  void refresh()

  return { items, loading, error, refresh }
}

// Sums per-project counts into the four registry stat cells.
export function aggregateCounts(projects: ProjectStats[]): RegistryCounts {
  const out: RegistryCounts = { total: 0, inProgress: 0, complete: 0, todo: 0 }
  for (const p of projects) {
    out.total += p.counts.total
    out.inProgress += p.counts['in-progress']
    out.complete += p.counts.complete
    out.todo += p.counts.todo
  }
  return out
}
