import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useTheme } from './useTheme'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
})

// jsdom doesn't implement matchMedia; stub it so systemDefault() can read
// the (fake) prefers-color-scheme signal.
function stubPrefersDark(dark: boolean) {
  window.matchMedia = vi.fn((query: string) => ({
    matches: dark,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
}

describe('useTheme', () => {
  it('exposes the three themes', () => {
    expect(useTheme().THEMES).toEqual(['paper', 'slate', 'ink'])
  })

  it('setTheme writes documentElement.dataset.theme and localStorage', () => {
    useTheme().setTheme('ink')
    expect(document.documentElement.dataset.theme).toBe('ink')
    expect(localStorage.getItem('mneme.theme')).toBe('ink')
  })

  it('setTheme updates the reactive current ref', () => {
    const t = useTheme()
    t.setTheme('slate')
    expect(t.current.value).toBe('slate')
  })

  it('init restores a stored theme', () => {
    localStorage.setItem('mneme.theme', 'slate')
    useTheme().init()
    expect(document.documentElement.dataset.theme).toBe('slate')
  })

  it('init ignores a bogus stored value and falls back to system pref', () => {
    localStorage.setItem('mneme.theme', 'chartreuse')
    stubPrefersDark(false)
    useTheme().init()
    expect(document.documentElement.dataset.theme).toBe('paper')
  })

  it('init falls back to ink when the system prefers dark', () => {
    stubPrefersDark(true)
    useTheme().init()
    expect(document.documentElement.dataset.theme).toBe('ink')
  })

  it('init falls back to paper when the system prefers light', () => {
    stubPrefersDark(false)
    useTheme().init()
    expect(document.documentElement.dataset.theme).toBe('paper')
  })
})
