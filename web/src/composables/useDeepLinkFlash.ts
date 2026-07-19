import { nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { flashElement } from '@/lib/flash'

// Reveal the entity a deep-link points at. When a list view is opened with
// `?flash=<id>`, scroll the matching row into view and flash it — reusing the
// same highlight the live-update layer uses. The id is matched against both
// `[data-flash-id]` (internal id, from search deep-links) and `[data-ref-id]`
// (public id, from a pasted mneme:// reference). `ready` reports when the list
// has rendered its rows (e.g. `() => !loading.value`) so we wait for async data
// before looking for the target row.
export function useDeepLinkFlash(ready: () => boolean): void {
  const route = useRoute()

  async function reveal() {
    const target = route.query.flash
    if (typeof target !== 'string' || target === '' || !ready()) return
    await nextTick() // let the list render (or re-render after loading)
    const sel = escapeAttr(target)
    const el = document.querySelector(`[data-flash-id="${sel}"], [data-ref-id="${sel}"]`)
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
