import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StatsRow from './StatsRow.vue'

describe('StatsRow', () => {
  it('renders the four stat cells with labels and values', () => {
    const w = mount(StatsRow, {
      props: { counts: { total: 8, inProgress: 2, complete: 3, todo: 1 } },
    })
    const cells = w.findAll('[data-test="stat-cell"]')
    expect(cells).toHaveLength(4)
    expect(cells[0].text()).toContain('total')
    expect(cells[0].text()).toContain('8')
    expect(cells[1].text()).toContain('in progress')
    expect(cells[1].text()).toContain('2')
    expect(cells[2].text()).toContain('complete')
    expect(cells[2].text()).toContain('3')
    expect(cells[3].text()).toContain('todo')
    expect(cells[3].text()).toContain('1')
  })
})
