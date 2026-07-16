import { ref, watch, type Ref } from 'vue'
import { listEnv, type EnvEntry } from '@/api/env'
import type { RefreshOptions } from './refresh'

export interface UseEnvResult {
  items: Ref<EnvEntry[]>
  loading: Ref<boolean>
  error: Ref<Error | null>
  refresh: (opts?: RefreshOptions) => Promise<void>
}

// useEnv loads one project's env entries, refetching whenever the project
// ref changes. Entries arrive key-ordered from the API. An empty project
// (none selected yet) yields an empty list and no request.
export function useEnv(project: Ref<string>): UseEnvResult {
  const items = ref<EnvEntry[]>([])
  const loading = ref(false)
  const error = ref<Error | null>(null)

  async function refresh(opts?: RefreshOptions) {
    if (!project.value) {
      items.value = []
      return
    }
    const silent = opts?.silent ?? false
    if (!silent) {
      loading.value = true
      error.value = null
    }
    try {
      items.value = await listEnv(project.value)
      if (silent) error.value = null // recovered — show fresh content
    } catch (err) {
      if (silent) return // best-effort: keep the current list visible
      error.value = err instanceof Error ? err : new Error(String(err))
    } finally {
      if (!silent) loading.value = false
    }
  }

  // Wrap refresh so a project change never leaks the watch args in as opts.
  watch(project, () => refresh(), { immediate: true })

  return { items, loading, error, refresh }
}
