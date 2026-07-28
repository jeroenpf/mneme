import { onBeforeUnmount, ref } from 'vue'

// A now-timestamp ref that ticks every intervalMs so relative-time labels
// don't go stale in a long-open tab. Call from component setup (the cleanup
// hook needs an instance); one caller per view is enough — pass the ref down.
export function useNow(intervalMs = 60_000) {
  const now = ref(Date.now())
  const id = setInterval(() => {
    now.value = Date.now()
  }, intervalMs)
  onBeforeUnmount(() => clearInterval(id))
  return now
}
