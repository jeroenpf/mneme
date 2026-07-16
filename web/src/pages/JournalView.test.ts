import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { JournalEntry } from '@/api/journal'
import { listJournal } from '@/api/journal'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import JournalView from './JournalView.vue'

vi.mock('@/api/journal', () => ({ listJournal: vi.fn() }))
// Stub the live layer: it constructs a real EventSource (absent in jsdom).
// Its own behaviour is covered by useLiveRefresh.test.ts.
vi.mock('@/composables/useLiveRefresh', () => ({ useLiveRefresh: vi.fn() }))

const entry = (over: Partial<JournalEntry>): JournalEntry => ({
  id: over.id ?? 'e1',
  session_ref: '',
  summary: 's',
  accomplished: [],
  deferred: [],
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

const JournalEntryCardStub = defineComponent({
  props: { entry: { type: Object, required: true } },
  render() {
    return h('div', { class: 'entry-stub' }, (this.entry as JournalEntry).summary)
  },
})

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/memory', component: defineComponent({ render: () => h('div') }) },
      { path: '/decisions', component: defineComponent({ render: () => h('div') }) },
      { path: '/snippets', component: defineComponent({ render: () => h('div') }) },
      { path: '/journal', component: JournalView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/journal')
  await router.isReady()
  const w = mount(JournalView, {
    global: { plugins: [router], stubs: { JournalEntryCard: JournalEntryCardStub } },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listJournal).mockReset().mockResolvedValue([
    entry({ id: 'a', summary: 'Apollo pagination', project: 'apollo' }),
    entry({ id: 'b', summary: 'Mneme migration runner' }), // global
  ])
})

describe('JournalView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountView()
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('renders entry summaries', async () => {
    const w = await mountView()
    expect(w.text()).toContain('Apollo pagination')
    expect(w.text()).toContain('Mneme migration runner')
  })

  it('filters by project', async () => {
    const w = await mountView()
    await w.get('[data-test="project-filter"]').setValue('apollo')
    await flushPromises()
    expect(w.text()).toContain('Apollo pagination')
    expect(w.text()).not.toContain('Mneme migration runner')
  })

  it('subscribes to live journal events and targets the new entry', async () => {
    await mountView()
    expect(useLiveRefresh).toHaveBeenCalledWith('journal', expect.anything())
    const opts = vi.mocked(useLiveRefresh).mock.calls[0]![1]
    expect(opts.flashTarget?.({ type: 'journal', id: 'a' })).toBe('[data-flash-id="a"]')
  })
})
