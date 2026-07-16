import { ref, watch, type Ref } from 'vue'
import { getDocument } from '@/api/documents'
import type { Document } from '@/types'
import type { RefreshOptions } from './refresh'

export interface UseDocumentResult {
  doc: Ref<Document | null>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

export function useDocument(id: Ref<string>): UseDocumentResult {
  const doc = ref<Document | null>(null)
  const loading = ref(true)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      doc.value = await getDocument(id.value)
      if (silent) error.value = null // recovered — show fresh content
    } catch (err) {
      if (silent) return // best-effort: keep the current doc visible
      doc.value = null
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      if (!silent) loading.value = false
    }
  }

  // An id change is a full navigation — always a loud refresh (show loading,
  // clear any prior error), never silent.
  watch(id, () => refresh(), { immediate: true })

  return { doc, loading, error, refresh }
}
