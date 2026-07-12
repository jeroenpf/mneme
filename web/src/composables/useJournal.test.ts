import { describe, expect, it } from 'vitest'
import { groupJournal } from './useJournal'
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
