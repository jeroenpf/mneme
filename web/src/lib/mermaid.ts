// Lazy mermaid with the theme derived from tokens.css at runtime — the
// single-source-of-truth rule extends into the SVGs. The module import is
// memoised, but initialize() lives in applyTheme() so a runtime theme switch
// can re-initialize from fresh token values (a memoised init would freeze the
// SVG at whichever theme rendered first).
let mod: Promise<typeof import('mermaid').default> | null = null

function token(styles: CSSStyleDeclaration, name: string): string {
  return styles.getPropertyValue(name).trim()
}

function load() {
  return (mod ??= import('mermaid').then((m) => m.default))
}

export async function applyTheme() {
  const m = await load()
  const s = getComputedStyle(document.documentElement)
  m.initialize({
    startOnLoad: false,
    theme: 'base',
    darkMode: token(s, '--mermaid-dark') === '1',
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
}

export async function renderDiagram(id: string, code: string): Promise<string> {
  const m = await load()
  const { svg } = await m.render(id, code)
  return svg
}
