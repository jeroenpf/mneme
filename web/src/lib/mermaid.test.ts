import { describe, it, expect, beforeEach, vi } from 'vitest'
import mermaid from 'mermaid'
import { applyTheme } from './mermaid'

// Spy on initialize so we can assert what theme values mermaid is (re)initialized with.
vi.mock('mermaid', () => ({
  default: { initialize: vi.fn(), render: vi.fn() },
}))

const root = document.documentElement

function setToken(name: string, value: string) {
  root.style.setProperty(name, value)
}

beforeEach(() => {
  root.removeAttribute('style')
  // restoreMocks clears spyOn state but not a factory-created vi.fn()'s call
  // history, so reset the initialize spy between cases explicitly.
  vi.mocked(mermaid.initialize).mockClear()
})

describe('mermaid applyTheme', () => {
  it('derives darkMode from the --mermaid-dark token (not a hardcoded constant)', async () => {
    setToken('--mermaid-dark', '1')
    setToken('--font-mono', 'MonoTest')

    await applyTheme()

    expect(mermaid.initialize).toHaveBeenCalledTimes(1)
    expect(vi.mocked(mermaid.initialize).mock.calls[0][0]).toMatchObject({
      darkMode: true,
      fontFamily: 'MonoTest',
    })
  })

  it('re-initializes with fresh token values when the theme flips (the caching bug)', async () => {
    setToken('--mermaid-dark', '0')
    await applyTheme()

    setToken('--mermaid-dark', '1')
    await applyTheme()

    const calls = vi.mocked(mermaid.initialize).mock.calls
    expect(calls).toHaveLength(2)
    expect(calls[0][0]).toMatchObject({ darkMode: false })
    expect(calls[1][0]).toMatchObject({ darkMode: true })
  })
})
