import { ref, watch, type Ref } from 'vue'
import { listEnv, type EnvEntry } from '@/api/env'

export interface UseEnvResult {
  items: Ref<EnvEntry[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: () => Promise<void>
}

// useEnv loads one project's env entries, refetching whenever the project
// ref changes. Entries arrive key-ordered from the API. An empty project
// (none selected yet) yields an empty list and no request.
export function useEnv(project: Ref<string>): UseEnvResult {
  const items = ref<EnvEntry[]>([])
  const loading = ref(false)
  const error = ref<Error | null>(null)

  async function refresh() {
    if (!project.value) {
      items.value = []
      return
    }
    loading.value = true
    error.value = null
    try {
      items.value = await listEnv(project.value)
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      loading.value = false
    }
  }

  watch(project, refresh, { immediate: true })

  return { items, loading, error, refresh }
}
