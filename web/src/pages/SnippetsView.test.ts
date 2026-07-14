import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Snippet } from '@/api/snippets'
import { listSnippets } from '@/api/snippets'
import SnippetsView from './SnippetsView.vue'

vi.mock('@/api/snippets', () => ({ listSnippets: vi.fn() }))

const snip = (over: Partial<Snippet>): Snippet => ({
  id: over.id ?? 's1',
  title: 't',
  language: 'go',
  content: 'c',
  tags: [],
  description: '',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

// Stub the card so this test stays about filtering/grouping and never
// triggers the real MCode → Prism async import.
const SnippetCardStub = defineComponent({
  props: { snippet: { type: Object, required: true } },
  render() {
    return h('div', { class: 'snippet-stub' }, (this.snippet as Snippet).title)
  },
})

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/memory', component: defineComponent({ render: () => h('div') }) },
      { path: '/decisions', component: defineComponent({ render: () => h('div') }) },
      { path: '/snippets', component: SnippetsView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/snippets')
  await router.isReady()
  const w = mount(SnippetsView, {
    global: { plugins: [router], stubs: { SnippetCard: SnippetCardStub } },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listSnippets).mockReset().mockResolvedValue([
    snip({ id: 'a', title: 'Cursor pagination', project: 'apollo', language: 'typescript', tags: ['pagination'] }),
    snip({ id: 'b', title: 'Errgroup fan-out', language: 'go', tags: ['concurrency'] }), // global
  ])
})

describe('SnippetsView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountView()
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('renders snippet titles', async () => {
    const w = await mountView()
    expect(w.text()).toContain('Cursor pagination')
    expect(w.text()).toContain('Errgroup fan-out')
  })

  it('filters by project', async () => {
    const w = await mountView()
    await w.get('[data-test="project-filter"]').setValue('apollo')
    await flushPromises()
    expect(w.text()).toContain('Cursor pagination')
    expect(w.text()).not.toContain('Errgroup fan-out')
  })

  it('filters by language', async () => {
    const w = await mountView()
    await w.get('[data-test="language-filter"]').setValue('go')
    await flushPromises()
    expect(w.text()).toContain('Errgroup fan-out')
    expect(w.text()).not.toContain('Cursor pagination')
  })

  it('filters by tag', async () => {
    const w = await mountView()
    await w.get('[data-test="tag-filter"]').setValue('pagination')
    await flushPromises()
    expect(w.text()).toContain('Cursor pagination')
    expect(w.text()).not.toContain('Errgroup fan-out')
  })
})
