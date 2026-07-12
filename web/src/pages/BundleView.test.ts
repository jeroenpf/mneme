import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { listProjects } from '@/api/projects'
import { getBundle, type Bundle } from '@/api/bundle'
import BundleView from './BundleView.vue'

vi.mock('@/api/projects', () => ({ listProjects: vi.fn() }))
vi.mock('@/api/bundle', () => ({ getBundle: vi.fn() }))

const makeBundle = (over: Partial<Bundle> = {}): Bundle => ({
  project: 'mneme',
  memory: { db: 'postgres' },
  active_plan: null,
  decisions: [],
  snippets: [],
  journal: [],
  markdown: '# Context bundle — mneme\n\n## Memory\n- **db**: postgres\n',
  ...over,
})

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/bundle', component: BundleView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/bundle')
  await router.isReady()
  const w = mount(BundleView, { global: { plugins: [router] } })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listProjects).mockReset().mockResolvedValue([
    { slug: 'mneme', name: 'Mneme' } as never,
    { slug: 'hyperion', name: 'Hyperion' } as never,
  ])
  vi.mocked(getBundle).mockReset().mockResolvedValue(makeBundle())
})

describe('BundleView', () => {
  it('lists projects and shows the empty state initially', async () => {
    const w = await mountView()
    expect(w.find('[data-test="empty"]').exists()).toBe(true)
    const opts = w.get('[data-test="project-select"]').findAll('option')
    expect(opts.map((o) => o.text())).toContain('Mneme')
  })

  it('loads and renders the digest when a project is chosen', async () => {
    const w = await mountView()
    await w.get('[data-test="project-select"]').setValue('mneme')
    await flushPromises()
    expect(getBundle).toHaveBeenCalledWith('mneme', undefined)
    expect(w.get('[data-test="digest"]').text()).toContain('Context bundle — mneme')
  })
})
