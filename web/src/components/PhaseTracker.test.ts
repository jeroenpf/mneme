import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PhaseTracker from './PhaseTracker.vue'

const phases = [
  { title: 'Scaffolding', status: 'done' as const },
  { title: 'Registry', status: 'wip' as const },
  { title: 'Viewer', status: 'todo' as const },
]

describe('PhaseTracker', () => {
  it('renders a connected spine node per phase with done/current/todo states', () => {
    const w = mount(PhaseTracker, { props: { phases } })
    expect(w.text()).toContain('1/3')
    const rows = w.findAll('[data-test="phase-row"]')
    expect(rows).toHaveLength(3)
    // status → spine state class (wip maps to "current" to match the mockup)
    expect(rows[0].classes()).toContain('done')
    expect(rows[1].classes()).toContain('current')
    expect(rows[2].classes()).toContain('todo')
    // the done node carries a check; each phase shows a status tag
    expect(rows[0].find('.node .check').exists()).toBe(true)
    expect(rows[1].find('.node .check').exists()).toBe(false)
    expect(rows[0].find('.p-tag').text()).toBe('done')
    expect(rows[1].find('.p-tag').text()).toBe('in progress')
    expect(rows[2].find('.p-tag').text()).toBe('todo')
  })
})
