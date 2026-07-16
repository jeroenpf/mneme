import { describe, expect, it, vi } from 'vitest'
import * as snippetsApi from '@/api/snippets'
import { groupSnippets, useSnippets } from './useSnippets'
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

describe('useSnippets silent refresh', () => {
  it('swaps items without ever toggling loading', async () => {
    const spy = vi.spyOn(snippetsApi, 'listSnippets').mockResolvedValue([snip({ id: 'a' })])
    const { items, loading, refresh } = useSnippets()
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue([snip({ id: 'a' }), snip({ id: 'b' })])
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → list stays mounted
    await p
    expect(items.value).toHaveLength(2)
    expect(loading.value).toBe(false)
  })
})
