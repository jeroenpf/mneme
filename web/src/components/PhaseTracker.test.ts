import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PhaseTracker from './PhaseTracker.vue'

const phases = [
  { title: 'Scaffolding', status: 'done' as const },
  { title: 'Registry', status: 'wip' as const },
  { title: 'Viewer', status: 'todo' as const },
]

describe('PhaseTracker', () => {
  it('renders a row per phase with its status pip and a done counter', () => {
    const w = mount(PhaseTracker, { props: { phases } })
    expect(w.text()).toContain('1/3')
    const rows = w.findAll('[data-test="phase-row"]')
    expect(rows).toHaveLength(3)
    expect(rows[0].find('.pip').classes()).toContain('pip-done')
    expect(rows[1].find('.pip').classes()).toContain('pip-wip')
    expect(rows[2].find('.pip').classes()).toContain('pip-todo')
    expect(rows[1].classes()).toContain('wip')
  })
})
