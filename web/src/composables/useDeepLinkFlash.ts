import { nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { flashElement } from '@/lib/flash'

// Reveal the entity a search deep-link points at. When a list view is opened
// with `?flash=<id>`, scroll the matching `[data-flash-id]` row into view and
// flash it — reusing the same highlight the live-update layer uses. `ready`
// reports when the list has rendered its rows (e.g. `() => !loading.value`) so
// we wait for async data before looking for the target row.
export function useDeepLinkFlash(ready: () => boolean): void {
  const route = useRoute()

  async function reveal() {
    const target = route.query.flash
    if (typeof target !== 'string' || target === '' || !ready()) return
    await nextTick() // let the list render (or re-render after loading)
    const el = document.querySelector(`[data-flash-id="${escapeAttr(target)}"]`)
    if (!el) return
    el.scrollIntoView({ block: 'center' })
    flashElement(el)
  }

  // immediate covers a cached/instant load; the ready watch covers async load;
  // the query watch covers navigating between deep-links without a remount.
  watch(ready, reveal, { immediate: true })
  watch(() => route.query.flash, reveal)
}

// CSS.escape is absent under jsdom; fall back to escaping the attribute
// selector's special characters for our id/key values.
function escapeAttr(v: string): string {
  const g = globalThis as { CSS?: { escape?: (s: string) => string } }
  return g.CSS?.escape ? g.CSS.escape(v) : v.replace(/["\\\]]/g, '\\$&')
}
