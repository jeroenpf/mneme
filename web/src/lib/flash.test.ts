import { afterEach, describe, expect, it, vi } from 'vitest'
import { flashElement } from './flash'

// jsdom has no matchMedia; stub it so flashElement can read the
// prefers-reduced-motion query. `reduce` toggles the returned `matches`.
function stubReducedMotion(reduce: boolean) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: reduce,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

describe('flashElement', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('no-ops on a null or undefined target', () => {
    stubReducedMotion(false)
    expect(() => flashElement(null)).not.toThrow()
    expect(() => flashElement(undefined)).not.toThrow()
  })

  it('adds the mn-flash class to the element', () => {
    stubReducedMotion(false)
    const el = document.createElement('div')
    flashElement(el)
    expect(el.classList.contains('mn-flash')).toBe(true)
  })

  it('does not flash under prefers-reduced-motion', () => {
    stubReducedMotion(true)
    const el = document.createElement('div')
    flashElement(el)
    expect(el.classList.contains('mn-flash')).toBe(false)
  })
})
