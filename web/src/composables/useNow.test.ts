import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, type Ref } from 'vue'
import { useNow } from './useNow'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

// useNow registers a cleanup hook, so it must run inside component setup.
function mountWithNow(intervalMs?: number) {
  let now!: Ref<number>
  const w = mount(
    defineComponent({
      setup() {
        now = useNow(intervalMs)
        return () => h('div')
      },
    }),
  )
  return { w, now }
}

describe('useNow', () => {
  it('starts at the current time and ticks every interval', () => {
    const start = Date.now()
    const { now } = mountWithNow(60_000)
    expect(now.value).toBe(start)
    vi.advanceTimersByTime(60_000)
    expect(now.value).toBe(start + 60_000)
  })

  it('stops ticking after unmount', () => {
    const { w, now } = mountWithNow(60_000)
    w.unmount()
    const frozen = now.value
    vi.advanceTimersByTime(120_000)
    expect(now.value).toBe(frozen)
  })
})
