import { describe, expect, it } from 'vitest'
import { renderInline } from './markdown'

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
