import { ref } from 'vue'

export const THEMES = ['paper', 'slate', 'ink'] as const
export type Theme = (typeof THEMES)[number]

const KEY = 'mneme.theme'
// Module-level singleton: every useTheme() call shares one active theme.
const current = ref<Theme>('paper')

function systemDefault(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'ink' : 'paper'
}

export function useTheme() {
  function setTheme(t: Theme) {
    current.value = t
    document.documentElement.dataset.theme = t
    localStorage.setItem(KEY, t)
  }

  function init() {
    const stored = localStorage.getItem(KEY) as Theme | null
    setTheme(stored && THEMES.includes(stored) ? stored : systemDefault())
  }

  return { current, THEMES, setTheme, init }
}
