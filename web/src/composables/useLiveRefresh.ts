import { nextTick, onScopeDispose } from 'vue'
import { useEventStream, type LiveEvent } from './useEventStream'
import { flashElement } from '@/lib/flash'

export interface LiveRefreshOptions {
  refresh: () => unknown | Promise<unknown>
  match?: (ev: LiveEvent) => boolean // default: all events of this type
  flashTarget?: (ev: LiveEvent) => Element | string | null
  debounceMs?: number // coalesce bursts; default 120
}

// useLiveRefresh wires a mounted view to the live stream: on a matching event
// it debounce-calls refresh(), then flashes the changed element; it also
// refreshes once on reconnect (resync, no flash). Auto-cleans on unmount.
export function useLiveRefresh(type: string, opts: LiveRefreshOptions): void {
  const stream = useEventStream()
  const match = opts.match ?? (() => true)
  const debounceMs = opts.debounceMs ?? 120
  let timer: ReturnType<typeof setTimeout> | undefined
  let pending: LiveEvent | null = null

  async function fire(ev: LiveEvent | null) {
    await opts.refresh()
    if (ev && opts.flashTarget) {
      await nextTick()
      const t = opts.flashTarget(ev)
      flashElement(typeof t === 'string' ? document.querySelector(t) : t)
    }
  }
  const offEvent = stream.subscribe(type, (ev) => {
    if (!match(ev)) return
    pending = ev
    clearTimeout(timer)
    timer = setTimeout(() => {
      const e = pending
      pending = null
      void fire(e)
    }, debounceMs)
  })
  const offReconnect = stream.onReconnect(() => {
    void fire(null)
  })
  onScopeDispose(() => {
    clearTimeout(timer)
    offEvent()
    offReconnect()
  })
}
