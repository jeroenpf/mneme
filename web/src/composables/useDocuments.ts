import { ref, watch, type Ref } from 'vue'
import { listDocuments } from '@/api/documents'
import type { Document, DocumentFilter } from '@/types'
import type { RefreshOptions } from './refresh'

export interface UseDocumentsResult {
  items: Ref<Document[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useDocuments(filter: Ref<DocumentFilter>): UseDocumentsResult {
  const items = ref<Document[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      const res = await listDocuments(filter.value)
      items.value = res.items
      if (silent) error.value = null // recovered — show fresh content
    } catch (err) {
      if (silent) return // best-effort: keep the current list visible
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      if (!silent) loading.value = false
    }
  }

  // Wrap refresh so a filter change never leaks the watch args in as opts.
  watch(filter, () => refresh(), { immediate: true, deep: true })

  return { items, loading, error, refresh }
}
