import { describe, expect, it } from 'vitest'
import { phasesFromMeta } from './phases'

describe('phasesFromMeta', () => {
  it('extracts well-formed phases', () => {
    expect(
      phasesFromMeta({
        phases: [
          { title: 'Scaffolding', status: 'done' },
          { title: 'Registry', status: 'wip' },
          { title: 'Viewer', status: 'todo' },
        ],
      }),
    ).toEqual([
      { title: 'Scaffolding', status: 'done' },
      { title: 'Registry', status: 'wip' },
      { title: 'Viewer', status: 'todo' },
    ])
  })

  it('coerces unknown status to todo and drops junk entries', () => {
    expect(
      phasesFromMeta({ phases: [{ title: 'A', status: 'later' }, 'junk', { status: 'done' }] }),
    ).toEqual([{ title: 'A', status: 'todo' }])
  })

  it('returns [] for missing/undefined meta or phases', () => {
    expect(phasesFromMeta(undefined)).toEqual([])
    expect(phasesFromMeta({})).toEqual([])
    expect(phasesFromMeta({ phases: 'nope' })).toEqual([])
  })
})
