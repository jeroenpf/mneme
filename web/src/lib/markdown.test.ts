import { describe, expect, it } from 'vitest'
import { renderInline, renderParagraphs } from './markdown'

describe('renderInline', () => {
  it('renders bold, italic, and inline code', () => {
    expect(renderInline('**bold** and *italic* and `code`')).toBe(
      '<strong>bold</strong> and <em>italic</em> and <code>code</code>',
    )
  })

  it('renders links with href', () => {
    expect(renderInline('[spec](/doc/spec)')).toBe('<a href="/doc/spec">spec</a>')
  })

  it('passes plain text through', () => {
    expect(renderInline('just words')).toBe('just words')
  })

  it('escapes raw html instead of interpreting it', () => {
    const out = renderInline('<script>alert(1)</script> & <b>x</b>')
    expect(out).not.toContain('<script>')
    expect(out).not.toContain('<b>')
    expect(out).toContain('&lt;script&gt;')
    expect(out).toContain('&amp;')
  })

  it('keeps angle brackets inside code spans displayable', () => {
    expect(renderInline('`<component :is>`')).toBe('<code>&lt;component :is&gt;</code>')
  })

  it('does not expand block-level markdown', () => {
    expect(renderInline('# not a heading')).toBe('# not a heading')
  })

  it('maps known :icon: shortcodes to glyphs and leaves unknown ones alone', () => {
    expect(renderInline('done :check: next :arrow:')).toBe('done ✓ next →')
    expect(renderInline(':nosuchicon:')).toBe(':nosuchicon:')
  })

  it('returns empty string for undefined/null/empty', () => {
    expect(renderInline(undefined)).toBe('')
    expect(renderInline(null)).toBe('')
    expect(renderInline('')).toBe('')
  })
})

describe('renderParagraphs', () => {
  it('returns an empty array for undefined/null/empty', () => {
    expect(renderParagraphs(undefined)).toEqual([])
    expect(renderParagraphs(null)).toEqual([])
    expect(renderParagraphs('')).toEqual([])
  })

  it('renders a single paragraph as one entry', () => {
    expect(renderParagraphs('just **one** para')).toEqual(['just <strong>one</strong> para'])
  })

  it('splits on blank lines into separate paragraphs', () => {
    expect(renderParagraphs('first para\n\nsecond para')).toEqual(['first para', 'second para'])
  })

  it('collapses a single newline within a paragraph (soft wrap, not a break)', () => {
    expect(renderParagraphs('one line\nsame para')).toEqual(['one line\nsame para'])
  })

  it('does not infer breaks from inline bold labels — only blank lines split', () => {
    // A single-line run of "**Label:**" clauses is one paragraph; the renderer
    // never guesses that a bold label starts a new line.
    expect(
      renderParagraphs('**Files:** a.go, b.go. **Outcome:** it works. **AC:** tests green.'),
    ).toEqual([
      '<strong>Files:</strong> a.go, b.go. <strong>Outcome:</strong> it works. <strong>AC:</strong> tests green.',
    ])
  })

  it('splits a blank-line-separated labeled run into one paragraph each', () => {
    expect(
      renderParagraphs('**Files:** a.go, b.go.\n\n**Outcome:** it works.\n\n**AC:** tests green.'),
    ).toEqual([
      '<strong>Files:</strong> a.go, b.go.',
      '<strong>Outcome:</strong> it works.',
      '<strong>AC:</strong> tests green.',
    ])
  })
})
