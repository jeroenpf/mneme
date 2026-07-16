import { describe, expect, it, vi } from 'vitest'
import * as solutionsApi from '@/api/solutions'
import { groupSolutions, useSolutions } from './useSolutions'
import type { Solution } from '@/api/solutions'

const sol = (over: Partial<Solution>): Solution => ({
  id: Math.random().toString(36).slice(2),
  error_description: 'e',
  solution: 's',
  tags: [],
  source_url: '',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('groupSolutions', () => {
  it('buckets by project with global first, newest-first within a bucket', () => {
    const groups = groupSolutions([
      sol({ project: 'apollo', error_description: 'older', created_at: '2026-07-01T00:00:00Z' }),
      sol({ project: 'apollo', error_description: 'newer', created_at: '2026-07-05T00:00:00Z' }),
      sol({ error_description: 'global one', created_at: '2026-07-02T00:00:00Z' }),
    ])
    expect(groups.map((g) => g.project)).toEqual(['', 'apollo'])
    expect(groups[1].solutions.map((s) => s.error_description)).toEqual(['newer', 'older'])
  })
})

describe('useSolutions silent refresh', () => {
  it('swaps items without ever toggling loading', async () => {
    const spy = vi.spyOn(solutionsApi, 'listSolutions').mockResolvedValue([sol({ id: 'a' })])
    const { items, loading, refresh } = useSolutions()
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue([sol({ id: 'a' }), sol({ id: 'b' })])
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → list stays mounted
    await p
    expect(items.value).toHaveLength(2)
    expect(loading.value).toBe(false)
  })
})
