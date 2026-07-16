import { Marked } from 'marked'

// Inline-only markdown for content/title/description strings. Block
// structure comes from typed components, never from markdown — this
// renders emphasis, code spans, and links, nothing else.
const marked = new Marked({ gfm: true, breaks: false })

// The codespan tokenizer re-escapes unconditionally (not entity-aware),
// which would double-escape the pre-escaping below and show the reader
// literal "&lt;". Undo exactly that one escape level; the pre-escape
// already guarantees no live HTML can survive in token.text.
marked.use({
  renderer: {
    codespan({ text }) {
      return `<code>${text.replace(/&amp;/g, '&')}</code>`
    },
  },
})

// The plan's :icon: shortcodes. Deliberately tiny — glyphs, not an icon
// library, so the output stays a plain HTML string.
const ICONS: Record<string, string> = {
  check: '✓',
  x: '✕',
  warn: '▲',
  info: 'ℹ',
  note: '◈',
  arrow: '→',
}

function escapeHtml(src: string): string {
  return src
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function renderInline(src: string | undefined | null): string {
  if (!src) return ''
  const iconed = src.replace(/:([a-z-]+):/g, (m, name: string) => ICONS[name] ?? m)
  // Pre-escaping makes raw-HTML tokens impossible; marked's escaper is
  // entity-aware so it won't double-escape what we produced here.
  return (marked.parseInline(escapeHtml(iconed)) as string).trim()
}

// renderParagraphs splits a prose field (text/section content, subphase
// description) into paragraphs, each rendered as inline markdown. Block
// STRUCTURE still comes only from typed blocks — this just lets one prose
// field hold several paragraphs, which is what prose is.
//
// A paragraph break is a blank line (the authoring convention: separate
// paragraphs with one blank line, not a separate block per paragraph). As a
// migration aid it also breaks before an inline bold field-label —
// "**Files:** … **Outcome:** …" — that trails text on the same line, so a
// description authored as one run of labeled clauses still renders one line
// each without needing a rewrite.
export function renderParagraphs(src: string | undefined | null): string[] {
  if (!src) return []
  const withBreaks = src.replace(/(\S)[ \t]+(\*\*[^*\n]+:\*\*)/g, '$1\n\n$2')
  return withBreaks
    .split(/\n[ \t]*\n/)
    .map((para) => renderInline(para))
    .filter(Boolean)
}
