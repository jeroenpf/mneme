import { describe, expect, it } from 'vitest'
import { groupSnippets } from './useSnippets'
import type { Snippet } from '@/api/snippets'

const snip = (over: Partial<Snippet>): Snippet => ({
  id: Math.random().toString(36).slice(2),
  title: 't',
  language: 'go',
  content: 'c',
  tags: [],
  description: '',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('groupSnippets', () => {
  it('buckets by project with global first, newest-first within a bucket', () => {
    const groups = groupSnippets([
      snip({ project: 'apollo', title: 'older', created_at: '2026-07-01T00:00:00Z' }),
      snip({ project: 'apollo', title: 'newer', created_at: '2026-07-05T00:00:00Z' }),
      snip({ title: 'global one', created_at: '2026-07-02T00:00:00Z' }),
    ])
    expect(groups.map((g) => g.project)).toEqual(['', 'apollo'])
    expect(groups[1].snippets.map((s) => s.title)).toEqual(['newer', 'older'])
  })
})
