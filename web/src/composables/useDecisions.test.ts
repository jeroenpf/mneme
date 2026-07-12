import { describe, expect, it } from 'vitest'
import { groupDecisions } from './useDecisions'
import type { Decision } from '@/api/decisions'

const dec = (over: Partial<Decision>): Decision => ({
  id: Math.random().toString(36).slice(2),
  title: 't',
  decision: 'd',
  rationale: '',
  alternatives: '',
  consequences: '',
  status: 'accepted',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('groupDecisions', () => {
  it('buckets by project with global first, newest-first within a bucket', () => {
    const groups = groupDecisions([
      dec({ project: 'apollo', title: 'older', created_at: '2026-07-01T00:00:00Z' }),
      dec({ project: 'apollo', title: 'newer', created_at: '2026-07-05T00:00:00Z' }),
      dec({ title: 'global one', created_at: '2026-07-02T00:00:00Z' }),
    ])
    expect(groups.map((g) => g.project)).toEqual(['', 'apollo'])
    expect(groups[1].decisions.map((d) => d.title)).toEqual(['newer', 'older'])
  })
})
