import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MTaskList from './MTaskList.vue'

const tasks = [
  { id: 't1', title: 'Ship the `viewer`', done: true, tags: ['vue'] },
  { id: 't2', title: 'Phase tracker', done: false, content: 'Built from `meta.phases[]`.' },
]

describe('MTaskList', () => {
  it('renders a row per task with done state, inline md, content, and tags', () => {
    const w = mount(MTaskList, { props: { title: 'Open follow-ups', tasks } })
    expect(w.text()).toContain('Open follow-ups')
    const rows = w.findAll('li')
    expect(rows).toHaveLength(2)
    expect(rows[0].find('.box').classes()).toContain('done')
    expect(rows[0].html()).toContain('<code>viewer</code>')
    expect(rows[0].text()).toContain('#vue')
    expect(rows[1].find('.box').classes()).not.toContain('done')
    expect(rows[1].html()).toContain('<code>meta.phases[]</code>')
  })

  it('renders nothing without tasks', () => {
    expect(mount(MTaskList, { props: {} }).find('ul').exists()).toBe(false)
  })
})
