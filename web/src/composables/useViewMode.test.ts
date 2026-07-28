import { beforeEach, describe, expect, it } from 'vitest'
import { useViewMode } from './useViewMode'

beforeEach(() => {
  localStorage.clear()
  // The singleton survives between tests; init() with a clean store resets it.
  useViewMode().init()
})

describe('useViewMode', () => {
  it('defaults to cards', () => {
    expect(useViewMode().view.value).toBe('cards')
  })

  it('setView updates the reactive ref and localStorage', () => {
    const m = useViewMode()
    m.setView('list')
    expect(m.view.value).toBe('list')
    expect(localStorage.getItem('mneme.registry-view')).toBe('list')
  })

  it('init restores a stored view', () => {
    localStorage.setItem('mneme.registry-view', 'list')
    const m = useViewMode()
    m.init()
    expect(m.view.value).toBe('list')
  })

  it('init ignores a bogus stored value and falls back to cards', () => {
    localStorage.setItem('mneme.registry-view', 'carousel')
    const m = useViewMode()
    m.init()
    expect(m.view.value).toBe('cards')
  })
})
