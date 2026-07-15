import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createEventStream } from './useEventStream'

// jsdom has no EventSource. FakeES records instances and lets the test drive
// open/error/message callbacks the composable installs.
class FakeES {
  static instances: FakeES[] = []
  url: string
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null

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
  close() {}
}

describe('createEventStream', () => {
  beforeEach(() => {
    FakeES.instances = []
    vi.stubGlobal('EventSource', FakeES)
  })
  afterEach(() => vi.unstubAllGlobals())

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
})
