import { ref, type Ref } from 'vue'
import { listJournal, type JournalEntry } from '@/api/journal'

export interface ProjectJournal {
  project: string // '' for global
  entries: JournalEntry[]
}

const newestFirst = (a: JournalEntry, b: JournalEntry) => b.created_at.localeCompare(a.created_at)

// groupJournal buckets a flat list by project (global = ''), each bucket
// sorted newest-first. Buckets are ordered global-first, then project slug.
export function groupJournal(entries: JournalEntry[]): ProjectJournal[] {
  const byProject = new Map<string, JournalEntry[]>()
  for (const e of entries) {
    const key = e.project ?? ''
    const arr = byProject.get(key) ?? []
    arr.push(e)
    byProject.set(key, arr)
  }
  const groups: ProjectJournal[] = [...byProject.entries()].map(([project, es]) => ({
    project,
    entries: [...es].sort(newestFirst),
  }))
  groups.sort((a, b) => {
    if (a.project === '') return -1
    if (b.project === '') return 1
    return a.project.localeCompare(b.project)
  })
  return groups
}

export interface UseJournalResult {
  items: Ref<JournalEntry[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

export function useJournal(): UseJournalResult {
  const items = ref<JournalEntry[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      items.value = await listJournal()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  void refresh()

  return { items, loading, error, refresh }
}
