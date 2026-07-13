import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { search, type SearchHit } from '@/api/search'
import SearchView from './SearchView.vue'

vi.mock('@/api/search', () => ({ search: vi.fn() }))

const hits: SearchHit[] = [
  { type: 'decisions', id: 'd1', title: 'use zigbee2mqtt', excerpt: 'adopt <<zigbee>>2mqtt', project: 'apollo', score: 0.9, updated_at: '2026-07-01T00:00:00Z' },
  { type: 'journal', id: 'j1', title: 'zigbee swap done', excerpt: '<<zigbee>> swap', project: 'apollo', score: 0.8, updated_at: '2026-07-02T00:00:00Z' },
]

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/doc/:id', component: defineComponent({ render: () => h('div') }) },
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
})

describe('SearchView', () => {
  it('fetches for the ?q= param and groups results by type', async () => {
    const w = await mountAt('/search?q=zigbee')
    expect(search).toHaveBeenCalledWith('zigbee')
    const text = w.text()
    expect(text).toContain('decisions')
    expect(text).toContain('use zigbee2mqtt')
    expect(text).toContain('journal')
  })

  it('shows an empty prompt with no query', async () => {
    vi.mocked(search).mockClear()
    const w = await mountAt('/search')
    expect(search).not.toHaveBeenCalled()
    expect(w.find('[data-test="empty"]').exists()).toBe(true)
  })
})
