import { describe, expect, it, vi } from 'vitest'
import { useToast } from './useToast'

describe('useToast', () => {
  it('adds a toast and auto-dismisses it after the ttl', () => {
    vi.useFakeTimers()
    const { toasts, toast } = useToast()
    const before = toasts.length
    const id = toast('Copied doc_1', 1000)
    expect(toasts.length).toBe(before + 1)
    expect(toasts.find((t) => t.id === id)?.message).toBe('Copied doc_1')
    vi.advanceTimersByTime(1000)
    expect(toasts.find((t) => t.id === id)).toBeUndefined()
    vi.useRealTimers()
  })

  it('dismiss removes a toast immediately', () => {
    const { toasts, toast, dismiss } = useToast()
    const before = toasts.length
    const id = toast('hi', 100000)
    expect(toasts.length).toBe(before + 1)
    dismiss(id)
    expect(toasts.length).toBe(before)
  })
})
