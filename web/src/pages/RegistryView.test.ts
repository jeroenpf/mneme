import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Document, ProjectStats } from '@/types'
import { listDocuments } from '@/api/documents'
import { listProjects } from '@/api/projects'
import RegistryView from './RegistryView.vue'

vi.mock('@/api/documents', () => ({ listDocuments: vi.fn() }))
vi.mock('@/api/projects', () => ({ listProjects: vi.fn() }))

const doc = (id: string, over: Partial<Document> = {}): Document => ({
  id,
  title: id,
  type: 'plan',
  status: 'todo',
  tags: [],
  meta: {},
  body: {},
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

const projects: ProjectStats[] = [
  {
    id: '1',
    name: 'Mneme',
    slug: 'mneme',
    created_at: '',
    counts: { todo: 1, 'in-progress': 2, complete: 3, blocked: 0, archived: 1, total: 7 },
  },
]

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: RegistryView },
      { path: '/doc/:id', component: defineComponent({ render: () => h('div') }) },
      { path: '/memory', component: defineComponent({ render: () => h('div') }) },
      { path: '/decisions', component: defineComponent({ render: () => h('div') }) },
      { path: '/snippets', component: defineComponent({ render: () => h('div') }) },
      { path: '/journal', component: defineComponent({ render: () => h('div') }) },
      { path: '/solutions', component: defineComponent({ render: () => h('div') }) },
    ],
  })
}

async function mountView(router: Router) {
  await router.push('/')
  await router.isReady()
  const wrapper = mount(RegistryView, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.mocked(listDocuments).mockReset().mockResolvedValue({
    items: [
      doc('alpha', { status: 'in-progress' }),
      doc('beta', { status: 'complete' }),
      doc('gamma', { status: 'archived' }),
    ],
    next_cursor: null,
  })
  vi.mocked(listProjects).mockReset().mockResolvedValue(projects)
})

describe('RegistryView', () => {
  it('renders a card per active doc and keeps archived collapsed', async () => {
    const w = await mountView(makeRouter())
    expect(w.find('[data-test="grid"]').findAll('.doc-card')).toHaveLength(2)
    expect(w.find('[data-test="archived-grid"]').exists()).toBe(false)
    const toggle = w.find('[data-test="archived-toggle"]')
    expect(toggle.text()).toContain('archived (1)')
    await toggle.trigger('click')
    expect(w.find('[data-test="archived-grid"]').findAll('.doc-card')).toHaveLength(1)
  })

  it('shows aggregated stats from the projects endpoint', async () => {
    const w = await mountView(makeRouter())
    const nums = w.findAll('[data-test="stat-cell"] .num').map((n) => n.text())
    expect(nums).toEqual(['7', '2', '3', '1'])
  })

  it('writes status pill clicks to the url and refetches', async () => {
    const router = makeRouter()
    const w = await mountView(router)
    const pill = w.findAll('button.pill').find((p) => p.text() === 'complete')
    await pill!.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.status).toBe('complete')
    expect(vi.mocked(listDocuments)).toHaveBeenLastCalledWith(
      expect.objectContaining({ status: 'complete' }),
    )
  })

  it('debounces search input into the url q param', async () => {
    vi.useFakeTimers()
    const router = makeRouter()
    const w = await mountView(router)
    await w.find('input[type="search"]').setValue('zigbee')
    expect(router.currentRoute.value.query.q).toBeUndefined()
    await vi.advanceTimersByTimeAsync(300)
    vi.useRealTimers()
    await flushPromises()
    expect(router.currentRoute.value.query.q).toBe('zigbee')
    expect(vi.mocked(listDocuments)).toHaveBeenLastCalledWith(
      expect.objectContaining({ q: 'zigbee' }),
    )
  })

  it('offers clear filters when a filtered view is empty', async () => {
    vi.mocked(listDocuments).mockResolvedValue({ items: [], next_cursor: null })
    const router = makeRouter()
    await router.push('/?status=blocked')
    await router.isReady()
    const w = mount(RegistryView, { global: { plugins: [router] } })
    await flushPromises()
    expect(w.find('[data-test="empty"]').text()).toContain('no documents match')
    await w.find('[data-test="empty"] button').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({})
  })
})
