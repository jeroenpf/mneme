import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { search, searchStatus, type SearchHit } from '@/api/search'
import { listProjects } from '@/api/projects'
import SearchView from './SearchView.vue'

vi.mock('@/api/search', () => ({ search: vi.fn(), searchStatus: vi.fn() }))
vi.mock('@/api/projects', () => ({ listProjects: vi.fn() }))

const hits: SearchHit[] = [
  { type: 'documents', id: 'road-p4', public_id: 'doc_p4', title: 'Retrieval plan', excerpt: '<<zigbee>> retrieval', project: 'apollo', score: 0.95, updated_at: '2026-07-03T00:00:00Z' },
  { type: 'decisions', id: 'd1', public_id: 'dec_z1', title: 'use zigbee2mqtt', excerpt: 'adopt <<zigbee>>2mqtt', project: 'apollo', score: 0.9, updated_at: '2026-07-01T00:00:00Z' },
  { type: 'journal', id: 'j1', public_id: 'jrnl_z1', title: 'zigbee swap done', excerpt: '<<zigbee>> swap', project: 'apollo', score: 0.8, updated_at: '2026-07-02T00:00:00Z' },
  { type: 'memory', id: 'm1', title: 'deploy-target', excerpt: 'runs on the <<zigbee>> host', project: 'apollo', score: 0.7, updated_at: '2026-07-04T00:00:00Z' },
]

const stub = defineComponent({ render: () => h('div') })

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: stub },
      { path: '/doc/:id', component: stub },
      { path: '/decisions', component: stub },
      { path: '/snippets', component: stub },
      { path: '/solutions', component: stub },
      { path: '/journal', component: stub },
      { path: '/memory', component: stub },
      { path: '/search', component: SearchView },
    ],
  })
}

async function mountAt(query: string) {
  const router = makeRouter()
  await router.push(query)
  await router.isReady()
  const w = mount(SearchView, { global: { plugins: [router] } })
  await flushPromises()
  return { w, router }
}

beforeEach(() => {
  vi.mocked(search).mockReset().mockResolvedValue(hits)
  vi.mocked(searchStatus).mockReset().mockResolvedValue({
    enabled: true,
    provider: { name: 'voyage', model: 'voyage-4-large', enabled: true },
    queue_depth: 0,
    items: [
      { type: 'documents', total: 5, embedded: 3, reconciled: 3, missing: 2, stale: 0, orphaned: 0, failed: 0 },
    ],
  })
  vi.mocked(listProjects)
    .mockReset()
    .mockResolvedValue([
      { slug: 'apollo', name: 'Apollo' } as never,
      { slug: 'mneme', name: 'Mneme' } as never,
    ])
})

describe('SearchView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const { w } = await mountAt('')
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('fetches for the ?q= param and groups results by type', async () => {
    const { w } = await mountAt('/search?q=zigbee')
    expect(search).toHaveBeenCalledWith('zigbee', { project: undefined })
    const text = w.text()
    expect(text).toContain('decisions')
    expect(text).toContain('use zigbee2mqtt')
    expect(text).toContain('journal')
  })

  it('deep-links every result type by its public id (memory by key)', async () => {
    const { w } = await mountAt('/search?q=zigbee')
    const hrefs = w.findAll('a.hit-title').map((a) => a.attributes('href'))
    // documents → the per-doc viewer; the rest → their list view + ?flash on
    // the stable public id, matching the row's data-ref-id.
    expect(hrefs).toContain('/doc/road-p4')
    expect(hrefs).toContain('/decisions?flash=dec_z1')
    expect(hrefs).toContain('/journal?flash=jrnl_z1')
    // memory has no public id — flashes on its key (the hit title).
    expect(hrefs).toContain('/memory?flash=deploy-target')
    expect(w.findAll('span.hit-title')).toHaveLength(0)
  })

  it('scopes the search to the selected project via the URL', async () => {
    const { w, router } = await mountAt('/search?q=zigbee')
    await w.get('[data-test="project-filter"]').setValue('apollo')
    await flushPromises()
    expect(router.currentRoute.value.query.project).toBe('apollo')
    expect(search).toHaveBeenLastCalledWith('zigbee', { project: 'apollo' })
  })

  it('reads an initial project scope from the URL', async () => {
    await mountAt('/search?q=zigbee&project=mneme')
    expect(search).toHaveBeenCalledWith('zigbee', { project: 'mneme' })
  })

  it('offers a copyable reference on results that carry a public id', async () => {
    const { w } = await mountAt('/search?q=zigbee')
    // documents/decisions/journal have public ids → a copy-ref control each;
    // the memory hit does not.
    expect(w.findAll('[data-test="copy-ref"]')).toHaveLength(3)
  })

  it('shows an empty prompt with no query', async () => {
    vi.mocked(search).mockClear()
    const { w } = await mountAt('/search')
    expect(search).not.toHaveBeenCalled()
    expect(w.find('[data-test="empty"]').exists()).toBe(true)
  })

  it('shows the embedding coverage line when enabled', async () => {
    const { w } = await mountAt('/search?q=zigbee')
    expect(searchStatus).toHaveBeenCalled()
    expect(w.text()).toMatch(/semantic|embedded/i)
  })
})
