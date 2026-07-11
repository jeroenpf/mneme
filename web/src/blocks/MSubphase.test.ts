import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MSubphase from './MSubphase.vue'

describe('MSubphase', () => {
  it('renders num badge, session, title, description, tasks, and children', () => {
    const w = mount(MSubphase, {
      props: {
        id: 'sp-1-7',
        num: '1.7',
        title: 'Document viewer',
        session: 7,
        description: 'Renders `body.sections`.',
        tasks: [{ id: 't1', title: 'shell', done: true }],
        children: [{ type: 'text', id: 'p1', content: 'note' }],
      },
    })
    expect(w.attributes('id')).toBe('sp-1-7')
    expect(w.text()).toContain('1.7')
    expect(w.text()).toContain('session 7')
    expect(w.text()).toContain('Document viewer')
    expect(w.html()).toContain('<code>body.sections</code>')
    expect(w.findAll('li')).toHaveLength(1)
    expect(w.text()).toContain('note')
  })

  it('omits session and task list when absent', () => {
    const w = mount(MSubphase, { props: { num: '2', title: 'Swap' } })
    expect(w.text()).not.toContain('session')
    expect(w.find('ul').exists()).toBe(false)
  })
})
