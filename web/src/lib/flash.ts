// flash.ts — briefly highlight what the agent just changed. No-op under
// prefers-reduced-motion or for a missing target. Uses a CSS class so theme
// tokens (--accent-dim) apply; restarts cleanly if re-flashed.
export function flashElement(el: Element | null | undefined): void {
  if (!el) return
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
  const node = el as HTMLElement
  node.classList.remove('mn-flash')
  void node.offsetWidth // reflow so the animation restarts
  node.classList.add('mn-flash')
  const done = () => {
    node.classList.remove('mn-flash')
    node.removeEventListener('animationend', done)
  }
  node.addEventListener('animationend', done)
}
