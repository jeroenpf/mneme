import { describe, expect, it, vi } from 'vitest'
import * as journalApi from '@/api/journal'
import { groupJournal, useJournal } from './useJournal'
import type { JournalEntry } from '@/api/journal'

const entry = (over: Partial<JournalEntry>): JournalEntry => ({
  id: Math.random().toString(36).slice(2),
  session_ref: '',
  summary: 's',
  accomplished: [],
  deferred: [],
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('groupJournal', () => {
  it('buckets by project with global first, newest-first within a bucket', () => {
    const groups = groupJournal([
      entry({ project: 'apollo', summary: 'older', created_at: '2026-07-01T00:00:00Z' }),
      entry({ project: 'apollo', summary: 'newer', created_at: '2026-07-05T00:00:00Z' }),
      entry({ summary: 'global one', created_at: '2026-07-02T00:00:00Z' }),
    ])
    expect(groups.map((g) => g.project)).toEqual(['', 'apollo'])
    expect(groups[1].entries.map((e) => e.summary)).toEqual(['newer', 'older'])
  })
})

describe('useJournal silent refresh', () => {
  it('swaps items without ever toggling loading', async () => {
    const spy = vi.spyOn(journalApi, 'listJournal').mockResolvedValue([entry({ id: 'a' })])
    const { items, loading, refresh } = useJournal()
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue([entry({ id: 'a' }), entry({ id: 'b' })])
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → list stays mounted
    await p
    expect(items.value).toHaveLength(2)
    expect(loading.value).toBe(false)
  })
})
