import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { getBundle, type Bundle } from '@/api/bundle'
import { listProjects } from '@/api/projects'
import ProjectHome from './ProjectHome.vue'

vi.mock('@/api/bundle', () => ({ getBundle: vi.fn() }))
vi.mock('@/api/projects', () => ({ listProjects: vi.fn() }))

const makeBundle = (over: Partial<Bundle> = {}): Bundle => ({
  project: 'mneme',
  memory: {},
  active_plan: {
    id: 'plan-master-roadmap',
    title: 'Master roadmap',
    status: 'in-progress',
    active_phase: 'Finish the roadmap',
    phase_current: 6,
    phase_total: 6,
  },
  plan_stats: { total: 4, in_progress: 1, todo: 1, complete: 2 },
  next_tasks: [
    { id: 's6-t6', title: 'Workflow UI + operational hardening', phase: 'Finish the roadmap' },
    { id: 's6-t7', title: 'Ship it', phase: 'Finish the roadmap' },
  ],
  blockers: [{ id: 'plan-blocked-thing', title: 'Waiting on cert renewal' }],
  deferred: [],
  decisions: [
    {
      id: 'dec-1',
      public_id: 'dec_abc123',
      title: 'Use pgx',
      status: 'accepted',
      rationale: 'raw SQL over ORM',
    },
  ],
  snippets: [],
  journal: [
    {
      id: 'jrnl-1',
      public_id: 'jrnl_xyz789',
      session_ref: 'stage-6',
      summary: 'Finished P6, starting P8',
      deferred: ['REST history endpoints', 'embedding status UI'],
    },
  ],
  markdown: '# Context bundle — mneme',
  ...over,
})

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/project/:slug', component: ProjectHome, props: true },
      { path: '/doc/:id', component: defineComponent({ render: () => h('div') }) },
      { path: '/decisions', component: defineComponent({ render: () => h('div') }) },
      { path: '/journal', component: defineComponent({ render: () => h('div') }) },
    ],
  })
}

async function mountHome(slug = 'mneme') {
  const router = makeRouter()
  await router.push(`/project/${slug}`)
  await router.isReady()
  const w = mount(ProjectHome, {
    props: { slug },
    global: { plugins: [router] },
    attachTo: document.body,
  })
  await flushPromises()
  return { w, router }
}

beforeEach(() => {
  vi.mocked(getBundle).mockReset().mockResolvedValue(makeBundle())
  vi.mocked(listProjects)
    .mockReset()
    .mockResolvedValue([
      { slug: 'mneme', name: 'Mneme' } as never,
      { slug: 'hyperion', name: 'Hyperion' } as never,
    ])
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ProjectHome', () => {
  it('loads the bundle for the route slug and shows the project name', async () => {
    const { w } = await mountHome('mneme')
    expect(getBundle).toHaveBeenCalledWith('mneme')
    expect(w.get('[data-test="project-name"]').text()).toContain('mneme')
  })

  it('shows current work: the active plan title and active phase', async () => {
    const { w } = await mountHome()
    const cw = w.get('[data-test="current-work"]')
    expect(cw.text()).toContain('Master roadmap')
    expect(cw.text()).toContain('Finish the roadmap')
    expect(cw.attributes('href')).toContain('/doc/plan-master-roadmap')
  })

  it('lists next tasks as deep links into the plan at the task anchor', async () => {
    const { w } = await mountHome()
    const tasks = w.findAll('[data-test="next-task"]')
    expect(tasks).toHaveLength(2)
    expect(tasks[0].text()).toContain('Workflow UI + operational hardening')
    expect(tasks[0].get('a').attributes('href')).toContain('/doc/plan-master-roadmap#s6-t6')
  })

  it('lists blockers linking to the blocked document', async () => {
    const { w } = await mountHome()
    const blockers = w.findAll('[data-test="blocker"]')
    expect(blockers).toHaveLength(1)
    expect(blockers[0].text()).toContain('Waiting on cert renewal')
    expect(blockers[0].get('a').attributes('href')).toContain('/doc/plan-blocked-thing')
  })

  it('shows recent decisions with status and rationale', async () => {
    const { w } = await mountHome()
    const decisions = w.findAll('[data-test="decision"]')
    expect(decisions).toHaveLength(1)
    expect(decisions[0].text()).toContain('Use pgx')
    expect(decisions[0].text()).toContain('accepted')
    expect(decisions[0].text()).toContain('raw SQL over ORM')
  })

  it('shows the handoff: the last session summary and its deferred work', async () => {
    const { w } = await mountHome()
    const handoff = w.get('[data-test="handoff"]')
    expect(handoff.text()).toContain('Finished P6, starting P8')
    expect(handoff.text()).toContain('REST history endpoints')
    expect(handoff.text()).toContain('embedding status UI')
  })

  it('falls back to a plan-stats message when no plan is in progress', async () => {
    vi.mocked(getBundle).mockResolvedValue(
      makeBundle({
        active_plan: null,
        next_tasks: [],
        plan_stats: { total: 3, in_progress: 0, todo: 0, complete: 3 },
      }),
    )
    const { w } = await mountHome()
    expect(w.find('[data-test="current-work"]').exists()).toBe(false)
    expect(w.get('[data-test="no-active-plan"]').text().toLowerCase()).toContain('complete')
  })

  it('switches project via the picker, routing to its home', async () => {
    const { w, router } = await mountHome('mneme')
    await w.get('[data-test="project-picker"]').setValue('hyperion')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/project/hyperion')
  })
})
