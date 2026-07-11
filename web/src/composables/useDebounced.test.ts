import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useDebounced } from './useDebounced'

describe('useDebounced', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('starts with the source value', () => {
    expect(useDebounced(ref('a'), 300).value).toBe('a')
  })

  it('trails the source by the delay', async () => {
    const source = ref('a')
    const out = useDebounced(source, 300)
    source.value = 'b'
    await nextTick()
    expect(out.value).toBe('a')
    vi.advanceTimersByTime(300)
    expect(out.value).toBe('b')
  })

  it('collapses rapid changes into the last value', async () => {
    const source = ref('a')
    const out = useDebounced(source, 300)
    source.value = 'b'
    await nextTick()
    vi.advanceTimersByTime(200)
    source.value = 'c'
    await nextTick()
    vi.advanceTimersByTime(200)
    expect(out.value).toBe('a')
    vi.advanceTimersByTime(100)
    expect(out.value).toBe('c')
  })
})
