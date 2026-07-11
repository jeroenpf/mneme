// Lazy mermaid with the dark theme derived from tokens.css at runtime —
// the single-source-of-truth rule extends into the SVGs.
let mermaid: Promise<typeof import('mermaid').default> | null = null

function token(styles: CSSStyleDeclaration, name: string): string {
  return styles.getPropertyValue(name).trim()
}

function load() {
  mermaid ??= (async () => {
    const mod = (await import('mermaid')).default
    const s = getComputedStyle(document.documentElement)
    mod.initialize({
      startOnLoad: false,
      theme: 'base',
      darkMode: true,
      fontFamily: token(s, '--font-mono'),
      themeVariables: {
        background: token(s, '--bg'),
        primaryColor: token(s, '--bg-elevated'),
        primaryTextColor: token(s, '--text-primary'),
        primaryBorderColor: token(s, '--border-strong'),
        secondaryColor: token(s, '--bg-overlay'),
        secondaryBorderColor: token(s, '--border'),
        tertiaryColor: token(s, '--bg-surface'),
        tertiaryBorderColor: token(s, '--border-soft'),
        lineColor: token(s, '--text-muted'),
        textColor: token(s, '--text-secondary'),
        edgeLabelBackground: token(s, '--bg-elevated'),
        fontSize: '13px',
      },
    })
    return mod
  })()
  return mermaid
}

export async function renderDiagram(id: string, code: string): Promise<string> {
  const m = await load()
  const { svg } = await m.render(id, code)
  return svg
}
