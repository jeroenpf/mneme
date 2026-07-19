import { describe, expect, it, vi } from 'vitest'
import type { Router } from 'vue-router'
import { tryOpenRef } from './openRef'

function fakeRouter() {
  return { push: vi.fn() } as unknown as Router
}

describe('tryOpenRef', () => {
  it('navigates to a pasted mneme:// reference and reports handled', () => {
    const router = fakeRouter()
    expect(tryOpenRef(router, 'mneme://document/doc_1/task/s6-t1')).toBe(true)
    expect(router.push).toHaveBeenCalledWith({ path: '/doc/doc_1', hash: '#s6-t1' })
  })

  it('navigates to a bare public id', () => {
    const router = fakeRouter()
    expect(tryOpenRef(router, 'dec_1')).toBe(true)
    expect(router.push).toHaveBeenCalledWith({ path: '/decisions', query: { flash: 'dec_1' } })
  })

  it('returns false for a plain search query (caller falls back to search)', () => {
    const router = fakeRouter()
    expect(tryOpenRef(router, 'how do we do pagination')).toBe(false)
    expect(router.push).not.toHaveBeenCalled()
  })
})
