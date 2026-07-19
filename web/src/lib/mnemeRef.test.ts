import { describe, expect, it } from 'vitest'
import { formatRef, parseRef, publicIdKind, routeForRef, type MnemeRef } from './mnemeRef'

describe('formatRef', () => {
  it('formats top-level entity references', () => {
    expect(formatRef('document', 'doc_000000000000')).toBe('mneme://document/doc_000000000000')
    expect(formatRef('decision', 'dec_000000000000')).toBe('mneme://decision/dec_000000000000')
    expect(formatRef('project', 'prj_000000000000')).toBe('mneme://project/prj_000000000000')
  })

  it('nests block and task references under their owning document', () => {
    expect(formatRef('block', 'blk_111111111111', 'doc_000000000000')).toBe(
      'mneme://document/doc_000000000000/block/blk_111111111111',
    )
    expect(formatRef('task', 'sp-1-1', 'doc_000000000000')).toBe(
      'mneme://document/doc_000000000000/task/sp-1-1',
    )
  })
})

describe('parseRef', () => {
  it('parses top-level references', () => {
    expect(parseRef('mneme://document/doc_000000000000')).toEqual<MnemeRef>({
      kind: 'document',
      id: 'doc_000000000000',
    })
    expect(parseRef('mneme://solution/sol_00000000000a')).toEqual<MnemeRef>({
      kind: 'solution',
      id: 'sol_00000000000a',
    })
  })

  it('parses nested block and task references', () => {
    expect(parseRef('mneme://document/doc_000000000000/block/blk_111111111111')).toEqual<MnemeRef>({
      kind: 'block',
      id: 'blk_111111111111',
      docId: 'doc_000000000000',
    })
    // Permissive on the child id so semantic ids (pre-migration) still parse.
    expect(parseRef('mneme://document/doc_000000000000/task/s6-t1')).toEqual<MnemeRef>({
      kind: 'task',
      id: 's6-t1',
      docId: 'doc_000000000000',
    })
  })

  it('parses a bare top-level public id', () => {
    expect(parseRef('doc_000000000000')).toEqual<MnemeRef>({ kind: 'document', id: 'doc_000000000000' })
    expect(parseRef('  dec_000000000000  ')).toEqual<MnemeRef>({ kind: 'decision', id: 'dec_000000000000' })
  })

  it('round-trips formatRef output', () => {
    for (const ref of [
      formatRef('document', 'doc_000000000000'),
      formatRef('decision', 'dec_000000000000'),
      formatRef('block', 'blk_111111111111', 'doc_000000000000'),
      formatRef('task', 'task_111111111111', 'doc_000000000000'),
    ]) {
      const parsed = parseRef(ref)
      expect(parsed).not.toBeNull()
      expect(formatRef(parsed!.kind, parsed!.id, parsed!.docId)).toBe(ref)
    }
  })

  it('returns null for anything that is not a reference', () => {
    for (const bad of [
      '',
      '   ',
      'not a reference',
      'http://document/doc_000000000000',
      'mneme://banana/doc_000000000000',
      'blk_000000000000', // bare nested id needs its document
      'task_000000000000',
      'mneme://document', // missing id
      'mneme://document/doc_000000000000/comment/x', // unknown relation
      'mneme://project/prj_000000000000/block/blk_111111111111', // non-document owner
    ]) {
      expect(parseRef(bad)).toBeNull()
    }
  })
})

describe('routeForRef', () => {
  it('routes a document to its viewer by public id', () => {
    expect(routeForRef({ kind: 'document', id: 'doc_1' })).toEqual({ path: '/doc/doc_1' })
  })

  it('routes a block/task to the document with the child id as the hash', () => {
    expect(routeForRef({ kind: 'block', id: 'blk_9', docId: 'doc_1' })).toEqual({
      path: '/doc/doc_1',
      hash: '#blk_9',
    })
    expect(routeForRef({ kind: 'task', id: 's6-t1', docId: 'doc_1' })).toEqual({
      path: '/doc/doc_1',
      hash: '#s6-t1',
    })
  })

  it('routes knowledge entities to their list view with a flash query', () => {
    expect(routeForRef({ kind: 'decision', id: 'dec_1' })).toEqual({ path: '/decisions', query: { flash: 'dec_1' } })
    expect(routeForRef({ kind: 'snippet', id: 'snip_1' })).toEqual({ path: '/snippets', query: { flash: 'snip_1' } })
    expect(routeForRef({ kind: 'journal', id: 'jrnl_1' })).toEqual({ path: '/journal', query: { flash: 'jrnl_1' } })
    expect(routeForRef({ kind: 'solution', id: 'sol_1' })).toEqual({ path: '/solutions', query: { flash: 'sol_1' } })
  })

  it('routes a project to the registry', () => {
    expect(routeForRef({ kind: 'project', id: 'prj_1' })).toEqual({ path: '/' })
  })
})

describe('publicIdKind', () => {
  it('maps a public id to its kind by prefix', () => {
    expect(publicIdKind('doc_000000000000')).toBe('document')
    expect(publicIdKind('jrnl_000000000000')).toBe('journal')
    expect(publicIdKind('nope')).toBeNull()
    expect(publicIdKind('xyz_000000000000')).toBeNull()
  })
})
