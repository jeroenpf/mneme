import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SnippetCard from './SnippetCard.vue'
import type { Snippet } from '@/api/snippets'

const snip = (over: Partial<Snippet> = {}): Snippet => ({
  id: 's1',
  title: 'Cursor pagination',
  language: 'typescript',
  content: 'const x = 1',
  tags: ['pagination'],
  description: 'Keyset helper.',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('SnippetCard', () => {
  it('renders title, language, tags, description', async () => {
    const w = mount(SnippetCard, { props: { snippet: snip() } })
    await flushPromises()
    expect(w.text()).toContain('Cursor pagination')
    expect(w.get('[data-test="lang"]').text()).toBe('typescript')
    expect(w.get('[data-test="tag"]').text()).toContain('pagination')
    expect(w.text()).toContain('Keyset helper.')
  })

  it('copies content to the clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const w = mount(SnippetCard, { props: { snippet: snip({ content: 'copy me' }) } })
    await flushPromises()
    await w.get('[data-test="copy"]').trigger('click')
    expect(writeText).toHaveBeenCalledWith('copy me')
    vi.unstubAllGlobals()
  })
})
