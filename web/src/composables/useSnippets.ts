import { ref, type Ref } from 'vue'
import { listSnippets, type Snippet } from '@/api/snippets'
import type { RefreshOptions } from './refresh'

export interface ProjectSnippets {
  project: string // '' for global
  snippets: Snippet[]
}

const newestFirst = (a: Snippet, b: Snippet) => b.created_at.localeCompare(a.created_at)

// groupSnippets buckets a flat list by project (global = ''), each bucket
// sorted newest-first. Buckets are ordered global-first, then project slug.
export function groupSnippets(snippets: Snippet[]): ProjectSnippets[] {
  const byProject = new Map<string, Snippet[]>()
  for (const sn of snippets) {
    const key = sn.project ?? ''
    const arr = byProject.get(key) ?? []
    arr.push(sn)
    byProject.set(key, arr)
  }
  const groups: ProjectSnippets[] = [...byProject.entries()].map(([project, sns]) => ({
    project,
    snippets: [...sns].sort(newestFirst),
  }))
  groups.sort((a, b) => {
    if (a.project === '') return -1
    if (b.project === '') return 1
    return a.project.localeCompare(b.project)
  })
  return groups
}

export interface UseSnippetsResult {
  items: Ref<Snippet[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useSnippets(): UseSnippetsResult {
  const items = ref<Snippet[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      items.value = await listSnippets()
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
