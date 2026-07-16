import { describe, expect, it, vi } from 'vitest'
import * as decisionsApi from '@/api/decisions'
import { groupDecisions, useDecisions } from './useDecisions'
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

describe('useDecisions silent refresh', () => {
  it('swaps items without ever toggling loading', async () => {
    const spy = vi.spyOn(decisionsApi, 'listDecisions').mockResolvedValue([dec({ id: 'a' })])
    const { items, loading, refresh } = useDecisions()
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue([dec({ id: 'a' }), dec({ id: 'b' })])
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → list stays mounted
    await p
    expect(items.value).toHaveLength(2)
    expect(loading.value).toBe(false)
  })

  it('keeps the current list when a silent refetch fails', async () => {
    const spy = vi.spyOn(decisionsApi, 'listDecisions').mockResolvedValue([dec({ id: 'a' })])
    const { items, error, refresh } = useDecisions()
    await vi.waitFor(() => expect(items.value).toHaveLength(1))

    spy.mockRejectedValueOnce(new Error('boom'))
    await refresh({ silent: true })
    expect(items.value).toHaveLength(1) // retained, not blanked
    expect(error.value).toBeNull() // best-effort refetch surfaces no error
  })
})
