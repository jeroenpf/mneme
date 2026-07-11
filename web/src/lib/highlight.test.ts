import { describe, expect, it } from 'vitest'
import { highlightCode } from './highlight'

describe('highlightCode', () => {
  it('emits prism token markup for go', async () => {
    const out = await highlightCode('func main() {}', 'go')
    expect(out).toContain('token keyword')
    expect(out).toContain('func')
  })

  it('emits token markup for sql and yaml', async () => {
    expect(await highlightCode('SELECT 1 FROM docs;', 'sql')).toContain('token keyword')
    expect(await highlightCode('channel: 20', 'yaml')).toContain('token')
  })

  it('resolves the ts alias', async () => {
    expect(await highlightCode('const x: string = "y"', 'ts')).toContain('token')
  })

  it('escapes and passes through unknown or missing languages', async () => {
    expect(await highlightCode('<raw> & text', 'cobol')).toBe('&lt;raw&gt; &amp; text')
    expect(await highlightCode('plain', undefined)).toBe('plain')
  })
})
