import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { search, searchStatus, type SearchHit } from '@/api/search'
import SearchView from './SearchView.vue'

vi.mock('@/api/search', () => ({ search: vi.fn(), searchStatus: vi.fn() }))

const hits: SearchHit[] = [
  { type: 'documents', id: 'road-p4', title: 'Retrieval plan', excerpt: '<<zigbee>> retrieval', project: 'apollo', score: 0.95, updated_at: '2026-07-03T00:00:00Z' },
  { type: 'decisions', id: 'd1', title: 'use zigbee2mqtt', excerpt: 'adopt <<zigbee>>2mqtt', project: 'apollo', score: 0.9, updated_at: '2026-07-01T00:00:00Z' },
  { type: 'journal', id: 'j1', title: 'zigbee swap done', excerpt: '<<zigbee>> swap', project: 'apollo', score: 0.8, updated_at: '2026-07-02T00:00:00Z' },
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
  return w
}

beforeEach(() => {
  vi.mocked(search).mockReset().mockResolvedValue(hits)
  vi.mocked(searchStatus).mockReset().mockResolvedValue({
    enabled: true,
    items: [{ type: 'documents', embedded: 3, total: 5 }],
  })
})

describe('SearchView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountAt('')
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('fetches for the ?q= param and groups results by type', async () => {
    const w = await mountAt('/search?q=zigbee')
    expect(search).toHaveBeenCalledWith('zigbee')
    const text = w.text()
    expect(text).toContain('decisions')
    expect(text).toContain('use zigbee2mqtt')
    expect(text).toContain('journal')
  })

  it('deep-links every result type to its viewer', async () => {
    const w = await mountAt('/search?q=zigbee')
    const hrefs = w.findAll('a.hit-title').map((a) => a.attributes('href'))
    // documents → the per-doc viewer; the rest → their list view + ?flash row.
    expect(hrefs).toContain('/doc/road-p4')
    expect(hrefs).toContain('/decisions?flash=d1')
    expect(hrefs).toContain('/journal?flash=j1')
    // memory flashes on its key (which is the hit title), not its uuid.
    expect(hrefs).toContain('/memory?flash=deploy-target')
    // Every hit is now a link — no plain-span fallbacks remain.
    expect(w.findAll('span.hit-title')).toHaveLength(0)
  })

  it('shows an empty prompt with no query', async () => {
    vi.mocked(search).mockClear()
    const w = await mountAt('/search')
    expect(search).not.toHaveBeenCalled()
    expect(w.find('[data-test="empty"]').exists()).toBe(true)
  })

  it('shows the embedding coverage line when enabled', async () => {
    const w = await mountAt('/search?q=zigbee')
    expect(searchStatus).toHaveBeenCalled()
    expect(w.text()).toMatch(/semantic|embedded/i)
  })
})
