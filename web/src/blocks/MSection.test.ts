import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MSection from './MSection.vue'

describe('MSection', () => {
  it('renders an anchored section with an mn-h2 title and dispatched children', () => {
    const w = mount(MSection, {
      props: {
        id: 'overview',
        title: 'Over**view**',
        children: [{ type: 'text', id: 'p1', content: 'hello' }],
      },
    })
    expect(w.attributes('id')).toBe('overview')
    expect(w.classes()).toContain('mn-anchor')
    const h = w.find('h2')
    expect(h.classes()).toContain('mn-h2')
    expect(h.html()).toContain('<strong>view</strong>')
    expect(w.text()).toContain('hello')
  })

  it('demotes nested section headings to mn-h3', () => {
    const w = mount(MSection, {
      props: {
        id: 'outer',
        title: 'Outer',
        children: [{ type: 'section', id: 'inner', title: 'Inner', children: [] }],
      },
    })
    const inner = w.find('#inner')
    expect(inner.find('h3').classes()).toContain('mn-h3')
  })

  it('does not leak type as a DOM attribute through the dispatcher', () => {
    const w = mount(MSection, {
      props: { id: 's', title: 'S', children: [{ type: 'text', id: 'p', content: 'x' }] },
    })
    expect(w.find('p').attributes('type')).toBeUndefined()
  })
})
