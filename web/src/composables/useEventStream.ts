import { ref, type Ref } from 'vue'

export type LiveEvent = { type: string; id: string; project?: string; blockId?: string; op?: string }
type Handler = (ev: LiveEvent) => void
export interface EventStream {
  status: Ref<'connecting' | 'open' | 'closed'>
  subscribe: (type: string, fn: Handler) => () => void
  onReconnect: (fn: () => void) => () => void
}

const BASE = import.meta.env.VITE_API_URL ?? ''

// The server sends a `ping` event every ~20s. If none (nor any message)
// arrives within STALE_MS we treat the connection as dead and reconnect.
// WATCHDOG_MS is how often we check. STALE_MS is ~2 missed heartbeats, wide
// enough that a single delayed ping never triggers a false reconnect.
const WATCHDOG_MS = 15_000
const STALE_MS = 45_000

// createEventStream owns one EventSource, fanning messages out to type-keyed
// handlers. onReconnect fires on every (re)open AFTER the first, so views can
// resync state missed while disconnected.
//
// A liveness watchdog reconnects when the stream goes silent: EventSource's
// native reconnect only fires when the socket cleanly errors, but a half-open
// connection (Vite's dev proxy holding the socket after the backend drops, or
// real half-open TCP after laptop sleep/network loss) never errors — it just
// stops delivering. The heartbeat lets us detect and recover from that.
// Exported for tests.
export function createEventStream(url = `${BASE}/api/events`): EventStream {
  const status = ref<'connecting' | 'open' | 'closed'>('connecting')
  const handlers = new Map<string, Set<Handler>>()
  const reconnectFns = new Set<() => void>()
  let opened = false
  let es: EventSource
  let lastSeen = Date.now()
  const seen = () => {
    lastSeen = Date.now()
  }

  function connect() {
    es = new EventSource(url)
    es.onopen = () => {
      seen()
      status.value = 'open'
      if (opened) reconnectFns.forEach((fn) => fn())
      opened = true
    }
    es.onerror = () => {
      status.value = 'connecting'
    } // EventSource auto-reconnects on a clean error
    es.onmessage = (e: MessageEvent) => {
      seen()
      let ev: LiveEvent
      try {
        ev = JSON.parse(e.data as string)
      } catch {
        return
      }
      handlers.get(ev.type)?.forEach((fn) => fn(ev))
    }
    es.addEventListener('ping', seen) // heartbeat — liveness only, not an entity change
  }

  function checkLiveness() {
    if (Date.now() - lastSeen <= STALE_MS) return
    // Silent too long: assume half-open and force a fresh connection. Reset
    // lastSeen so we give the new socket a full window before trying again.
    status.value = 'connecting'
    es.close()
    seen()
    connect()
  }

  connect()
  setInterval(checkLiveness, WATCHDOG_MS)

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
