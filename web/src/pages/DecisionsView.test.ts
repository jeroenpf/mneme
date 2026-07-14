import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Decision } from '@/api/decisions'
import { listDecisions } from '@/api/decisions'
import DecisionsView from './DecisionsView.vue'

vi.mock('@/api/decisions', () => ({ listDecisions: vi.fn() }))

const dec = (over: Partial<Decision>): Decision => ({
  id: over.id ?? 'd1',
  title: 't',
  decision: 'd',
  rationale: '',
  alternatives: '',
  consequences: '',
  status: 'accepted',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/memory', component: defineComponent({ render: () => h('div') }) },
      { path: '/decisions', component: DecisionsView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/decisions')
  await router.isReady()
  const w = mount(DecisionsView, { global: { plugins: [router] } })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listDecisions).mockReset().mockResolvedValue([
    dec({ id: 'a', title: 'Use pgx', project: 'apollo', rationale: 'native types' }),
    dec({ id: 'b', title: 'Raw SQL only' }), // global
  ])
})

describe('DecisionsView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountView()
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('renders decision titles', async () => {
    const w = await mountView()
    expect(w.text()).toContain('Use pgx')
    expect(w.text()).toContain('Raw SQL only')
  })

  it('reveals rationale when a row is expanded', async () => {
    const w = await mountView()
    const row = w.get('[data-test="decision-a"]')
    expect(w.find('[data-test="detail-a"]').exists()).toBe(false)
    await row.get('[data-test="decision-toggle"]').trigger('click')
    await flushPromises()
    expect(w.get('[data-test="detail-a"]').text()).toContain('native types')
  })

  it('filters by project', async () => {
    const w = await mountView()
    await w.get('[data-test="project-filter"]').setValue('apollo')
    await flushPromises()
    expect(w.text()).toContain('Use pgx')
    expect(w.text()).not.toContain('Raw SQL only')
  })
})
