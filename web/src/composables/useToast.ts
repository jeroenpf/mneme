import { reactive, readonly, type DeepReadonly } from 'vue'

// A tiny app-wide toast queue for transient confirmations (e.g. "Copied
// reference"). State is module-level so any component can raise a toast and a
// single ToastHost renders them.
export interface Toast {
  id: number
  message: string
}

const state = reactive<{ items: Toast[] }>({ items: [] })
let seq = 0

export function useToast() {
  // toast enqueues a message and schedules its removal after ttl ms; returns
  // the id so callers can dismiss it early.
  function toast(message: string, ttl = 2200): number {
    const id = ++seq
    state.items.push({ id, message })
    if (ttl > 0) {
      setTimeout(() => dismiss(id), ttl)
    }
    return id
  }

  function dismiss(id: number): void {
    const i = state.items.findIndex((t) => t.id === id)
    if (i >= 0) state.items.splice(i, 1)
  }

  return { toasts: readonly(state).items as DeepReadonly<Toast[]>, toast, dismiss }
}
