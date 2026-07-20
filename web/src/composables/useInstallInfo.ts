import { ref, type Ref } from 'vue'
import { getInstall, type InstallInfo } from '@/api/install'

export interface UseInstallInfoResult {
  info: Ref<InstallInfo | null>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

// useInstallInfo loads the effective install facts (URL, MCP endpoint, storage,
// embeddings) once, immediately. The values are static for the server's
// lifetime, so there is nothing to re-fetch beyond a manual retry.
export function useInstallInfo(): UseInstallInfoResult {
  const info = ref<InstallInfo | null>(null)
  const loading = ref(false)
  const error = ref<Error | null>(null)

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      info.value = await getInstall()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  void refresh()

  return { info, loading, error, refresh }
}
