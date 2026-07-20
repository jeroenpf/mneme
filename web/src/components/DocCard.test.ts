import { describe, expect, it } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import type { Document } from '@/types'
import DocCard from './DocCard.vue'

function makeDoc(overrides: Partial<Document> = {}): Document {
  return {
    id: 'mneme-implementation',
    title: 'Mneme implementation',
    project: 'mneme',
    type: 'plan',
    status: 'in-progress',
    ticket: 'MN-1',
    repo: 'jeroenpf/mneme',
    tags: ['go', 'vue'],
    phase_current: 6,
    phase_total: 9,
    meta: { description: 'Phase plan for the core service.' },
    body: {},
    created_at: '2026-05-22T10:00:00Z',
    updated_at: '2026-07-11T10:00:00Z',
    ...overrides,
  }
}

function mountCard(doc: Document) {
  return mount(DocCard, {
    props: { doc },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('DocCard', () => {
  it('renders type, ticket, title, description, tags, and repo', () => {
    const w = mountCard(makeDoc())
    expect(w.text()).toContain('plan')
    expect(w.text()).toContain('MN-1')
    expect(w.text()).toContain('Mneme implementation')
    expect(w.text()).toContain('Phase plan for the core service.')
    expect(w.text()).toContain('#go')
    expect(w.text()).toContain('#vue')
    expect(w.find('[data-test="repo"]').text()).toContain('jeroenpf/mneme')
  })

  it('links to the document route', () => {
    const w = mountCard(makeDoc())
    expect(w.findComponent(RouterLinkStub).props('to')).toBe('/doc/mneme-implementation')
  })

  it('colors the status dot by status', () => {
    const w = mountCard(makeDoc({ status: 'blocked' }))
    expect(w.find('.status-dot').classes()).toContain('status-blocked')
  })

  it('renders one pip per phase, split into done/wip/todo', () => {
    const w = mountCard(makeDoc({ phase_current: 6, phase_total: 9 }))
    const pips = w.find('[data-test="pips"]').findAll('.pip')
    expect(pips).toHaveLength(9)
    expect(pips.filter((p) => p.classes().includes('pip-done'))).toHaveLength(5)
    expect(pips.filter((p) => p.classes().includes('pip-wip'))).toHaveLength(1)
    expect(w.find('[data-test="pips"]').text()).toContain('6/9')
  })

  it('omits optional rows when data is absent', () => {
    const w = mountCard(
      makeDoc({
        ticket: undefined,
        repo: undefined,
        tags: [],
        phase_current: undefined,
        phase_total: undefined,
        meta: {},
      }),
    )
    expect(w.find('[data-test="pips"]').exists()).toBe(false)
    expect(w.find('[data-test="repo"]').exists()).toBe(false)
    expect(w.text()).not.toContain('#')
  })
})
