import { apiDelete, apiGet, apiPut, buildQuery } from './client'

// Mirrors internal/models.EnvEntry. Non-secret project-scoped config.
export interface EnvEntry {
  id: string
  project: string
  key: string
  value: string
  description?: string
  updated_at: string
}

export function listEnv(project: string): Promise<EnvEntry[]> {
  return apiGet<{ items: EnvEntry[] }>(`/api/v1/env${buildQuery({ project })}`).then((r) => r.items)
}

export function setEnv(e: {
  project: string
  key: string
  value: string
  description?: string
}): Promise<EnvEntry> {
  const path = `/api/v1/env/${encodeURIComponent(e.key)}${buildQuery({ project: e.project })}`
  return apiPut<EnvEntry>(path, { value: e.value, description: e.description })
}

export function deleteEnv(e: { project: string; key: string }): Promise<void> {
  const path = `/api/v1/env/${encodeURIComponent(e.key)}${buildQuery({ project: e.project })}`
  return apiDelete(path)
}
