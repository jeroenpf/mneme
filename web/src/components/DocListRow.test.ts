import { describe, expect, it } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import type { Document } from '@/types'
import DocListRow from './DocListRow.vue'

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
    meta: {},
    body: {},
    created_at: '2026-05-22T10:00:00Z',
    updated_at: '2026-07-11T10:00:00Z',
    ...overrides,
  }
}

// Two hours after the default doc's updated_at.
const NOW = Date.parse('2026-07-11T12:00:00Z')

function mountRow(doc: Document) {
  return mount(DocListRow, {
    props: { doc, now: NOW },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('DocListRow', () => {
  it('shows a status badge carrying the status text and class', () => {
    const badge = mountRow(makeDoc()).find('[data-test="badge"]')
    expect(badge.text()).toBe('in-progress')
    expect(badge.classes()).toContain('status-in-progress')
  })

  it('renders type, title, ticket, and project', () => {
    const w = mountRow(makeDoc())
    expect(w.text()).toContain('plan')
    expect(w.text()).toContain('Mneme implementation')
    expect(w.text()).toContain('MN-1')
    expect(w.text()).toContain('mneme')
  })

  it('renders phase pips with the n/m label', () => {
    const w = mountRow(makeDoc())
    expect(w.find('[data-test="pips"]').findAll('.pip')).toHaveLength(9)
    expect(w.find('[data-test="pips"]').text()).toContain('6/9')
  })

  it('shows the relative age with the exact timestamp as tooltip', () => {
    const updated = mountRow(makeDoc()).find('[data-test="updated"]')
    expect(updated.text()).toBe('2h ago')
    expect(updated.attributes('title')).toContain('2026')
  })

  it('links to the document route', () => {
    const w = mountRow(makeDoc())
    expect(w.findComponent(RouterLinkStub).props('to')).toBe('/doc/mneme-implementation')
  })

  it('keeps all seven cells, empty, when optional fields are absent', () => {
    const w = mountRow(
      makeDoc({
        ticket: undefined,
        project: undefined,
        phase_current: undefined,
        phase_total: undefined,
      }),
    )
    expect(w.find('[data-test="pips"]').exists()).toBe(false)
    // Badge, type, title, pips cell, ticket cell, project cell, updated.
    expect(w.find('.doc-row').element.children).toHaveLength(7)
  })
})
