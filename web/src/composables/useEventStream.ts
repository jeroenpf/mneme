import { ref, type Ref } from 'vue'

export type LiveEvent = { type: string; id: string; project?: string; blockId?: string; op?: string }
type Handler = (ev: LiveEvent) => void
export interface EventStream {
  status: Ref<'connecting' | 'open' | 'closed'>
  subscribe: (type: string, fn: Handler) => () => void
  onReconnect: (fn: () => void) => () => void
}

const BASE = import.meta.env.VITE_API_URL ?? ''

// createEventStream owns one EventSource, fanning messages out to type-keyed
// handlers. onReconnect fires on every (re)open AFTER the first, so views can
// resync state missed while disconnected. Exported for tests.
export function createEventStream(url = `${BASE}/api/events`): EventStream {
  const status = ref<'connecting' | 'open' | 'closed'>('connecting')
  const handlers = new Map<string, Set<Handler>>()
  const reconnectFns = new Set<() => void>()
  let opened = false
  const es = new EventSource(url)
  es.onopen = () => {
    status.value = 'open'
    if (opened) reconnectFns.forEach((fn) => fn())
    opened = true
  }
  es.onerror = () => {
    status.value = 'connecting'
  } // EventSource auto-reconnects
  es.onmessage = (e: MessageEvent) => {
    let ev: LiveEvent
    try {
      ev = JSON.parse(e.data as string)
    } catch {
      return
    }
    handlers.get(ev.type)?.forEach((fn) => fn(ev))
  }
  return {
    status,
    subscribe(type, fn) {
      const set = handlers.get(type) ?? new Set<Handler>()
      set.add(fn)
      handlers.set(type, set)
      return () => set.delete(fn)
    },
    onReconnect(fn) {
      reconnectFns.add(fn)
      return () => reconnectFns.delete(fn)
    },
  }
}

let shared: EventStream | null = null
export function useEventStream(): EventStream {
  if (!shared) shared = createEventStream()
  return shared
}
