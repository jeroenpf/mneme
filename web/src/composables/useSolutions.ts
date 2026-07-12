import { ref, type Ref } from 'vue'
import { listSolutions, type Solution } from '@/api/solutions'

export interface ProjectSolutions {
  project: string // '' for global
  solutions: Solution[]
}

const newestFirst = (a: Solution, b: Solution) => b.created_at.localeCompare(a.created_at)

// groupSolutions buckets a flat list by project (global = ''), each bucket
// sorted newest-first. Buckets are ordered global-first, then project slug.
export function groupSolutions(solutions: Solution[]): ProjectSolutions[] {
  const byProject = new Map<string, Solution[]>()
  for (const s of solutions) {
    const key = s.project ?? ''
    const arr = byProject.get(key) ?? []
    arr.push(s)
    byProject.set(key, arr)
  }
  const groups: ProjectSolutions[] = [...byProject.entries()].map(([project, sols]) => ({
    project,
    solutions: [...sols].sort(newestFirst),
  }))
  groups.sort((a, b) => {
    if (a.project === '') return -1
    if (b.project === '') return 1
    return a.project.localeCompare(b.project)
  })
  return groups
}

export interface UseSolutionsResult {
  items: Ref<Solution[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

export function useSolutions(): UseSolutionsResult {
  const items = ref<Solution[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      items.value = await listSolutions()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  void refresh()

  return { items, loading, error, refresh }
}
