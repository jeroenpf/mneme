import { describe, expect, it, vi } from 'vitest'
import * as projectsApi from '@/api/projects'
import type { ProjectStats } from '@/types'
import { aggregateCounts, useProjects } from './useProjects'

function stats(slug: string, counts: Partial<ProjectStats['counts']> = {}): ProjectStats {
  return {
    id: slug,
    name: slug,
    slug,
    created_at: '',
    counts: { todo: 0, 'in-progress': 0, complete: 0, blocked: 0, archived: 0, total: 0, ...counts },
  }
}

describe('useProjects', () => {
  it('fetches on creation and exposes items', async () => {
    const spy = vi
      .spyOn(projectsApi, 'listProjects')
      .mockResolvedValue([stats('mneme', { total: 3 })])
    const { items, loading, error } = useProjects()
    expect(loading.value).toBe(true)
    await vi.waitFor(() => expect(loading.value).toBe(false))
    expect(items.value).toHaveLength(1)
    expect(error.value).toBeNull()
    expect(spy).toHaveBeenCalledOnce()
  })

  it('exposes error when the api throws', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockRejectedValueOnce(new Error('boom'))
    const { error, loading } = useProjects()
    await vi.waitFor(() => expect(loading.value).toBe(false))
    expect(error.value?.message).toBe('boom')
  })
})

describe('aggregateCounts', () => {
  it('sums the registry stat cells across projects', () => {
    const out = aggregateCounts([
      stats('a', { total: 4, 'in-progress': 2, complete: 1, todo: 1 }),
      stats('b', { total: 3, 'in-progress': 0, complete: 2, todo: 0 }),
    ])
    expect(out).toEqual({ total: 7, inProgress: 2, complete: 3, todo: 1 })
  })

  it('returns zeros for no projects', () => {
    expect(aggregateCounts([])).toEqual({ total: 0, inProgress: 0, complete: 0, todo: 0 })
  })
})
