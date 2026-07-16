import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createEventStream } from './useEventStream'

// jsdom has no EventSource. FakeES records instances and lets the test drive
// open/error/message/ping callbacks the composable installs.
class FakeES {
  static instances: FakeES[] = []
  url: string
  closed = false
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  listeners: Record<string, Set<(e: unknown) => void>> = {}

  constructor(url: string) {
    this.url = url
    FakeES.instances.push(this)
  }
  open() {
    this.onopen?.()
  }
  error() {
    this.onerror?.()
  }
  emit(payload: unknown) {
    this.onmessage?.({ data: JSON.stringify(payload) })
  }
  raw(data: string) {
    this.onmessage?.({ data })
  }
  addEventListener(type: string, fn: (e: unknown) => void) {
    ;(this.listeners[type] ??= new Set()).add(fn)
  }
  ping() {
    this.listeners['ping']?.forEach((fn) => fn({}))
  }
  close() {
    this.closed = true
  }
}

describe('createEventStream', () => {
  beforeEach(() => {
    FakeES.instances = []
    vi.stubGlobal('EventSource', FakeES)
    vi.useFakeTimers() // drive the liveness watchdog deterministically
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('opens exactly one EventSource at the given url', () => {
    createEventStream('/api/events')
    expect(FakeES.instances).toHaveLength(1)
    expect(FakeES.instances[0].url).toBe('/api/events')
  })

  it('dispatches a message only to handlers of its type', () => {
    const stream = createEventStream('/api/events')
    const es = FakeES.instances[0]
    const docs = vi.fn()
    const decisions = vi.fn()
    stream.subscribe('documents', docs)
    stream.subscribe('decisions', decisions)
    es.emit({ type: 'documents', id: 'd1' })
    expect(docs).toHaveBeenCalledWith({ type: 'documents', id: 'd1' })
    expect(decisions).not.toHaveBeenCalled()
  })

  it('stops delivery after unsubscribe', () => {
    const stream = createEventStream('/api/events')
    const es = FakeES.instances[0]
    const fn = vi.fn()
    const off = stream.subscribe('documents', fn)
    off()
    es.emit({ type: 'documents', id: 'd1' })
    expect(fn).not.toHaveBeenCalled()
  })

  it('ignores malformed JSON payloads', () => {
    const stream = createEventStream('/api/events')
    const es = FakeES.instances[0]
    const fn = vi.fn()
    stream.subscribe('documents', fn)
    es.raw('not json')
    expect(fn).not.toHaveBeenCalled()
  })

  it('tracks status across connecting/open/error', () => {
    const stream = createEventStream('/api/events')
    const es = FakeES.instances[0]
    expect(stream.status.value).toBe('connecting')
    es.open()
    expect(stream.status.value).toBe('open')
    es.error()
    expect(stream.status.value).toBe('connecting')
  })

  it('fires onReconnect only on the SECOND open (resync after a drop)', () => {
    const stream = createEventStream('/api/events')
    const es = FakeES.instances[0]
    const onReconnect = vi.fn()
    stream.onReconnect(onReconnect)
    es.open() // initial connect — not a reconnect
    expect(onReconnect).not.toHaveBeenCalled()
    es.error()
    es.open() // reconnect — resync
    expect(onReconnect).toHaveBeenCalledTimes(1)
  })

  it('a heartbeat keeps a quiet-but-healthy connection alive (no reconnect)', () => {
    createEventStream('/api/events')
    FakeES.instances[0].open()
    // Past the stale threshold in wall-clock, but pings keep arriving.
    for (let elapsed = 0; elapsed < 90_000; elapsed += 15_000) {
      vi.advanceTimersByTime(15_000)
      FakeES.instances.at(-1)!.ping()
    }
    expect(FakeES.instances).toHaveLength(1) // never reconnected
  })

  it('force-reconnects a half-open connection that goes silent past the threshold', () => {
    const stream = createEventStream('/api/events')
    const es0 = FakeES.instances[0]
    es0.open()
    expect(FakeES.instances).toHaveLength(1)

    // No ping, no message: after the stale window the watchdog opens a fresh
    // EventSource and closes the dead one.
    vi.advanceTimersByTime(60_000)
    expect(FakeES.instances.length).toBeGreaterThanOrEqual(2)
    expect(es0.closed).toBe(true)
    expect(stream.status.value).toBe('connecting')
  })

  it('resyncs when the watchdog-forced reconnection opens', () => {
    const stream = createEventStream('/api/events')
    FakeES.instances[0].open()
    const onReconnect = vi.fn()
    stream.onReconnect(onReconnect)

    vi.advanceTimersByTime(60_000) // stale → new EventSource
    const es1 = FakeES.instances.at(-1)!
    es1.open() // the replacement connects
    expect(onReconnect).toHaveBeenCalledTimes(1)
    expect(stream.status.value).toBe('open')
  })
})
