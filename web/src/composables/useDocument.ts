import { ref, watch, type Ref } from 'vue'
import { getDocument } from '@/api/documents'
import type { Document } from '@/types'

export interface UseDocumentResult {
  doc: Ref<Document | null>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

export function useDocument(id: Ref<string>): UseDocumentResult {
  const doc = ref<Document | null>(null)
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      doc.value = await getDocument(id.value)
    } catch (err) {
      doc.value = null
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  watch(id, refresh, { immediate: true })

  return { doc, loading, error, refresh }
}
