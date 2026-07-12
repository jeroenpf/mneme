import { describe, expect, it } from 'vitest'
import { groupSolutions } from './useSolutions'
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
