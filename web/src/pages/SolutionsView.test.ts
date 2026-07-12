import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Solution } from '@/api/solutions'
import { listSolutions } from '@/api/solutions'
import SolutionsView from './SolutionsView.vue'

vi.mock('@/api/solutions', () => ({ listSolutions: vi.fn() }))

const sol = (over: Partial<Solution>): Solution => ({
  id: over.id ?? 's1',
  error_description: 'e',
  solution: 's',
  tags: [],
  source_url: '',
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

const SolutionCardStub = defineComponent({
  props: { solution: { type: Object, required: true } },
  render() {
    return h('div', { class: 'sol-stub' }, (this.solution as Solution).error_description)
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
      { path: '/journal', component: defineComponent({ render: () => h('div') }) },
      { path: '/solutions', component: SolutionsView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/solutions')
  await router.isReady()
  const w = mount(SolutionsView, {
    global: { plugins: [router], stubs: { SolutionCard: SolutionCardStub } },
  })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listSolutions).mockReset().mockResolvedValue([
    sol({ id: 'a', error_description: 'Apollo docker timeout', project: 'apollo' }),
    sol({ id: 'b', error_description: 'Global mDNS stall' }), // global
  ])
})

describe('SolutionsView', () => {
  it('renders solution error descriptions', async () => {
    const w = await mountView()
    expect(w.text()).toContain('Apollo docker timeout')
    expect(w.text()).toContain('Global mDNS stall')
  })

  it('filters by project', async () => {
    const w = await mountView()
    await w.get('[data-test="project-filter"]').setValue('apollo')
    await flushPromises()
    expect(w.text()).toContain('Apollo docker timeout')
    expect(w.text()).not.toContain('Global mDNS stall')
  })
})
