import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { flashElement } from '@/lib/flash'
import type { EventStream, LiveEvent } from './useEventStream'
import { useLiveRefresh } from './useLiveRefresh'

vi.mock('@/lib/flash', () => ({ flashElement: vi.fn() }))

// A controllable fake stream shared with the mocked useEventStream. The mock
// factory only dereferences these at call time (inside useLiveRefresh), so
// they are initialised before any test runs.
const handlers = new Map<string, Set<(ev: LiveEvent) => void>>()
const reconnectFns = new Set<() => void>()
const fakeStream: EventStream = {
  status: ref<'connecting' | 'open' | 'closed'>('open'),
  subscribe(type, fn) {
    const set = handlers.get(type) ?? new Set()
    set.add(fn)
    handlers.set(type, set)
    return () => set.delete(fn)
  },
  onReconnect(fn) {
    reconnectFns.add(fn)
    return () => reconnectFns.delete(fn)
  },
}
function emit(ev: LiveEvent) {
  handlers.get(ev.type)?.forEach((fn) => fn(ev))
}
function reconnect() {
  reconnectFns.forEach((fn) => fn())
}

vi.mock('./useEventStream', () => ({ useEventStream: () => fakeStream }))

describe('useLiveRefresh', () => {
  beforeEach(() => {
    handlers.clear()
    reconnectFns.clear()
  })
  afterEach(() => vi.clearAllMocks())

  it('refreshes then flashes the resolved target on a matching event', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const el = document.createElement('div')
    const scope = effectScope()
    scope.run(() => {
      useLiveRefresh('documents', { refresh, flashTarget: () => el, debounceMs: 0 })
    })
    emit({ type: 'documents', id: 'd1' })
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    await vi.waitFor(() => expect(flashElement).toHaveBeenCalledWith(el))
    scope.stop()
  })

  it('ignores events that fail the match predicate', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const scope = effectScope()
    scope.run(() => {
      useLiveRefresh('documents', { refresh, match: (ev) => ev.id === 'keep', debounceMs: 0 })
    })
    emit({ type: 'documents', id: 'other' })
    await new Promise((r) => setTimeout(r, 5))
    expect(refresh).not.toHaveBeenCalled()
    scope.stop()
  })

  it('coalesces a burst into a single refresh', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const scope = effectScope()
    scope.run(() => {
      useLiveRefresh('documents', { refresh, debounceMs: 20 })
    })
    emit({ type: 'documents', id: 'd1' })
    emit({ type: 'documents', id: 'd2' })
    emit({ type: 'documents', id: 'd3' })
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    scope.stop()
  })

  it('refreshes on reconnect without flashing (resync)', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const scope = effectScope()
    scope.run(() => {
      useLiveRefresh('documents', {
        refresh,
        flashTarget: () => document.createElement('div'),
        debounceMs: 0,
      })
    })
    reconnect()
    await vi.waitFor(() => expect(refresh).toHaveBeenCalledTimes(1))
    expect(flashElement).not.toHaveBeenCalled()
    scope.stop()
  })

  it('stops listening once its effect scope is disposed', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const scope = effectScope()
    scope.run(() => {
      useLiveRefresh('documents', { refresh, debounceMs: 0 })
    })
    scope.stop()
    emit({ type: 'documents', id: 'd1' })
    await new Promise((r) => setTimeout(r, 5))
    expect(refresh).not.toHaveBeenCalled()
  })
})
