import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { Document } from '@/types'
import MetaHeader from './MetaHeader.vue'

function makeDoc(overrides: Partial<Document> = {}): Document {
  return {
    id: 'mneme-implementation',
    title: 'Mneme **implementation**',
    project: 'mneme',
    type: 'plan',
    status: 'in-progress',
    ticket: 'MN-1',
    repo: 'jeroenpfeil/mneme',
    tags: ['go', 'vue'],
    phase_current: 6,
    phase_total: 9,
    meta: {
      description: 'Phase plan for the core service.',
      custom_fields: { Complexity: 'Medium' },
    },
    body: {},
    created_at: '2026-05-22T10:00:00Z',
    updated_at: '2026-07-11T10:00:00Z',
    ...overrides,
  }
}

describe('MetaHeader', () => {
  it('renders title (inline md), type, ticket, repo, tags, description', () => {
    const w = mount(MetaHeader, { props: { doc: makeDoc() } })
    expect(w.find('h1').html()).toContain('<strong>implementation</strong>')
    expect(w.text()).toContain('plan')
    expect(w.text()).toContain('MN-1')
    expect(w.text()).toContain('jeroenpfeil/mneme')
    expect(w.text()).toContain('#go')
    expect(w.text()).toContain('Phase plan for the core service.')
  })

  it('renders the meta grid cells only for present fields', () => {
    const w = mount(MetaHeader, { props: { doc: makeDoc() } })
    const cells = w.findAll('[data-test="meta-cell"]')
    const labels = cells.map((c) => c.find('.mn-label').text())
    expect(labels).toEqual(['project', 'ticket', 'phase', 'created', 'updated'])
    expect(cells[2].text()).toContain('6/9')
    expect(cells[3].text()).toContain('2026-05-22')
  })

  it('renders custom_fields as a key-value grid', () => {
    const w = mount(MetaHeader, { props: { doc: makeDoc() } })
    expect(w.find('dl').text()).toContain('Complexity')
  })

  it('omits everything optional on a sparse doc', () => {
    const w = mount(MetaHeader, {
      props: {
        doc: makeDoc({
          project: undefined,
          ticket: undefined,
          repo: undefined,
          tags: [],
          phase_current: undefined,
          phase_total: undefined,
          meta: {},
        }),
      },
    })
    const labels = w.findAll('[data-test="meta-cell"] .mn-label').map((c) => c.text())
    expect(labels).toEqual(['created', 'updated'])
    expect(w.find('dl').exists()).toBe(false)
    expect(w.text()).not.toContain('#')
  })
})
