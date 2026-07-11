import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Document } from '@/types'
import { getDocument } from '@/api/documents'
import DocumentView from './DocumentView.vue'

vi.mock('@/api/documents', () => ({ getDocument: vi.fn() }))
vi.mock('@/lib/mermaid', () => ({ renderDiagram: vi.fn().mockResolvedValue('<svg></svg>') }))

const doc: Document = {
  id: 'mneme-implementation',
  title: 'Mneme implementation',
  project: 'mneme',
  type: 'plan',
  status: 'in-progress',
  tags: ['go'],
  phase_current: 6,
  phase_total: 9,
  meta: {
    description: 'Phase plan.',
    phases: [
      { title: 'Scaffolding', status: 'done' },
      { title: 'Registry', status: 'wip' },
    ],
  },
  body: {
    sections: [
      {
        type: 'section',
        id: 'overview',
        title: 'Overview',
        children: [{ type: 'text', id: 'p1', content: 'hello **world**' }],
      },
      {
        type: 'subphase',
        id: 'sp-1-7',
        num: '1.7',
        title: 'Viewer',
        tasks: [{ id: 't1', title: 'shell', done: true }],
      },
    ],
  },
  created_at: '2026-05-22T10:00:00Z',
  updated_at: '2026-07-11T10:00:00Z',
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div', 'registry') }) },
      { path: '/doc/:id', component: DocumentView, props: true },
    ],
  })
}

async function mountView(router: Router, url = '/doc/mneme-implementation') {
  await router.push(url)
  await router.isReady()
  // attachTo: hash scrolling resolves targets via document.getElementById,
  // which only sees attached nodes.
  const wrapper = mount(DocumentView, {
    props: { id: 'mneme-implementation' },
    global: { plugins: [router] },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.mocked(getDocument).mockReset().mockResolvedValue(doc)
  Element.prototype.scrollIntoView = vi.fn()
})

afterEach(() => {
  // Attached wrappers persist in jsdom's shared document — clear so ids
  // from one test can't satisfy the next test's getElementById.
  document.body.innerHTML = ''
})

describe('DocumentView', () => {
  it('renders meta header, phase tracker, section nav, and dispatched blocks', async () => {
    const w = await mountView(makeRouter())
    expect(w.find('h1').text()).toContain('Mneme implementation')
    expect(w.findAll('[data-test="phase-row"]')).toHaveLength(2)
    expect(w.findAll('.secnav a').map((a) => a.attributes('href'))).toEqual([
      '#overview',
      '#sp-1-7',
    ])
    expect(w.find('#overview').exists()).toBe(true)
    expect(w.find('#overview').html()).toContain('<strong>world</strong>')
    expect(w.find('#sp-1-7').text()).toContain('1.7')
    expect(w.find('[data-test="doc-status"]').text()).toContain('in-progress')
  })

  it('scrolls to the hash target once the document has loaded', async () => {
    const w = await mountView(makeRouter(), '/doc/mneme-implementation#sp-1-7')
    await flushPromises()
    await w.vm.$nextTick()
    expect(Element.prototype.scrollIntoView).toHaveBeenCalled()
  })

  it('shows the error state with retry', async () => {
    vi.mocked(getDocument).mockRejectedValue(new Error('boom'))
    const w = await mountView(makeRouter())
    expect(w.find('[data-test="doc-error"]').text()).toContain('could not load document')
    vi.mocked(getDocument).mockResolvedValue(doc)
    await w.find('[data-test="doc-error"] button').trigger('click')
    await flushPromises()
    expect(w.find('h1').text()).toContain('Mneme implementation')
  })

  it('back link navigates to the registry when there is no history entry', async () => {
    const router = makeRouter()
    const w = await mountView(router)
    await w.find('[data-test="back"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('hides sidebar groups that have no data', async () => {
    vi.mocked(getDocument).mockResolvedValue({ ...doc, meta: {}, body: {} })
    const w = await mountView(makeRouter())
    expect(w.find('[data-test="phase-row"]').exists()).toBe(false)
    expect(w.find('.secnav').exists()).toBe(false)
  })
})
