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

  // Refetch only when the filter's *contents* change. Callers may pass a
  // computed that yields a fresh object on any upstream change (e.g. a
  // client-side status toggle in the registry); watching the serialized
  // value — not the reference — skips redundant fetches for value-equal
  // filters. The callback also never leaks the watch args in as opts.
  watch(
    () => JSON.stringify(filter.value),
    () => refresh(),
    { immediate: true },
  )

  return { items, loading, error, refresh }
}
