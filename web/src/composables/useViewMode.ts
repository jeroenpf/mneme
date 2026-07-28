import { ref } from 'vue'

export const VIEWS = ['cards', 'list'] as const
export type ViewMode = (typeof VIEWS)[number]

const KEY = 'mneme.registry-view'
// Module-level singleton: every useViewMode() call shares one active view.
const current = ref<ViewMode>('cards')

export function useViewMode() {
  function setView(v: ViewMode) {
    current.value = v
    localStorage.setItem(KEY, v)
  }

  function init() {
    const stored = localStorage.getItem(KEY) as ViewMode | null
    current.value = stored && VIEWS.includes(stored) ? stored : 'cards'
  }

  return { view: current, VIEWS, setView, init }
}
