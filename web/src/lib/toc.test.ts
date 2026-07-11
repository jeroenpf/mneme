import { describe, expect, it } from 'vitest'
import { sectionNavItems } from './toc'

describe('sectionNavItems', () => {
  it('lists top-level sections and subphases with ids, keeping order', () => {
    expect(
      sectionNavItems({
        sections: [
          { type: 'section', id: 'overview', title: 'Overview', children: [{ type: 'section', id: 'nested', title: 'Nested' }] },
          { type: 'callout', id: 'c1', content: 'not navigable' },
          { type: 'subphase', id: 'sp-1-7', num: '1.7', title: 'Viewer' },
          { type: 'section', title: 'No id — skipped' },
        ],
      }),
    ).toEqual([
      { id: 'overview', title: 'Overview', kind: 'section', num: undefined },
      { id: 'sp-1-7', title: 'Viewer', kind: 'subphase', num: '1.7' },
    ])
  })

  it('returns [] for empty/missing body', () => {
    expect(sectionNavItems(undefined)).toEqual([])
    expect(sectionNavItems({})).toEqual([])
  })
})
