import { ref, type Ref } from 'vue'
import { listProjects } from '@/api/projects'
import type { ProjectStats } from '@/types'

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
  refresh: () => Promise<void>
}

export function useProjects(): UseProjectsResult {
  const items = ref<ProjectStats[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      items.value = await listProjects()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
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
