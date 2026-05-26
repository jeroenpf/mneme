import { ref, watch, type Ref } from 'vue'
import { listDocuments } from '@/api/documents'
import type { Document, DocumentFilter } from '@/types'

export interface UseDocumentsResult {
  items: Ref<Document[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

export function useDocuments(filter: Ref<DocumentFilter>): UseDocumentsResult {
  const items = ref<Document[]>([])
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      const res = await listDocuments(filter.value)
      items.value = res.items
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  watch(filter, refresh, { immediate: true, deep: true })

  return { items, loading, error, refresh }
}
