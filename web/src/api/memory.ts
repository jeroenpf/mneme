import { apiDelete, apiGet, apiPut, buildQuery } from './client'

export type MemoryScope = 'global' | 'project' | 'area'

// Mirrors internal/models.Memory. project/area are absent for scopes
// that don't use them.
export interface MemoryEntry {
  id: string
  scope: MemoryScope
  project?: string
  area?: string
  key: string
  value: string
  updated_at: string
}

export interface MemoryTarget {
  scope: MemoryScope
  key: string
  project?: string
  area?: string
}

export function listMemory(): Promise<MemoryEntry[]> {
  return apiGet<{ items: MemoryEntry[] }>('/api/v1/memory').then((r) => r.items)
}

export function setMemory(e: MemoryTarget & { value: string }): Promise<MemoryEntry> {
  const path = `/api/v1/memory/${e.scope}/${encodeURIComponent(e.key)}${buildQuery({
    project: e.project,
    area: e.area,
  })}`
  return apiPut<MemoryEntry>(path, { value: e.value })
}

export function deleteMemory(e: MemoryTarget): Promise<void> {
  const path = `/api/v1/memory/${e.scope}/${encodeURIComponent(e.key)}${buildQuery({
    project: e.project,
    area: e.area,
  })}`
  return apiDelete(path)
}
