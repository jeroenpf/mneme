// Lazy Prism: nothing prism-related lands in the main chunk. First call
// pulls core + the 8 supported grammars; later calls reuse them.
const ALIASES: Record<string, string> = { ts: 'typescript', js: 'javascript', sh: 'bash' }

type PrismNS = typeof import('prismjs')

let prism: Promise<PrismNS> | null = null

function load(): Promise<PrismNS> {
  prism ??= (async () => {
    const g = globalThis as { Prism?: unknown }
    // Prism core reads this flag at import time — stops it scanning the DOM.
    if (!g.Prism) g.Prism = { manual: true }
    // prismjs is CJS (`export =`); interop differs between Vite and node,
    // so accept either a default export or the namespace itself.
    const imported = await import('prismjs')
    const mod = (imported as { default?: PrismNS }).default ?? imported
    // php extends markup-templating; import it first, the rest are independent.
    await import('prismjs/components/prism-markup-templating')
    await Promise.all([
      import('prismjs/components/prism-go'),
      import('prismjs/components/prism-sql'),
      import('prismjs/components/prism-bash'),
      import('prismjs/components/prism-yaml'),
      import('prismjs/components/prism-json'),
      import('prismjs/components/prism-typescript'),
      import('prismjs/components/prism-php'),
    ])
    return mod
  })()
  return prism
}

function escapeHtml(src: string): string {
  return src.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export async function highlightCode(code: string, lang?: string): Promise<string> {
  if (!lang) return escapeHtml(code)
  const Prism = await load()
  const name = ALIASES[lang] ?? lang
  const grammar = Prism.languages[name]
  if (!grammar) return escapeHtml(code)
  return Prism.highlight(code, grammar, name)
}
