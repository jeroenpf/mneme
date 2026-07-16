import { describe, expect, it } from 'vitest'
import { sectionNavItems } from './toc'

describe('sectionNavItems', () => {
  it('lists top-level sections and subphases with sequential display numbers, keeping order', () => {
    // Every navigable entry gets a zero-padded sequential number so the TOC
    // reads 01, 02, … in document order regardless of block type. A block's
    // own `num` (a subphase's real phase number) is not used for the TOC —
    // the body still renders that badge; the nav counts positionally.
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
      { id: 'overview', title: 'Overview', kind: 'section', num: '01' },
      { id: 'sp-1-7', title: 'Viewer', kind: 'subphase', num: '02' },
    ])
  })

  it('returns [] for empty/missing body', () => {
    expect(sectionNavItems(undefined)).toEqual([])
    expect(sectionNavItems({})).toEqual([])
  })
})
