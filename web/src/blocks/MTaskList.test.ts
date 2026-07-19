import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import MTaskList from './MTaskList.vue'
import { DOC_PUBLIC_ID } from '@/composables/useDocRef'

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

  it('gives each task row a DOM id matching the task id (block-flash target)', () => {
    const w = mount(MTaskList, { props: { tasks } })
    const rows = w.findAll('li')
    expect(rows[0].attributes('id')).toBe('t1')
    expect(rows[1].attributes('id')).toBe('t2')
  })

  it('offers a per-task copy-reference control when a document id is in scope', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    const w = mount(MTaskList, {
      props: { tasks },
      global: { provide: { [DOC_PUBLIC_ID as symbol]: ref('doc_000000000000') } },
    })
    const controls = w.findAll('[data-test="copy-ref"]')
    expect(controls).toHaveLength(2)
    await controls[0].trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('mneme://document/doc_000000000000/task/t1')
  })

  it('renders nothing without tasks', () => {
    expect(mount(MTaskList, { props: {} }).find('ul').exists()).toBe(false)
  })
})
