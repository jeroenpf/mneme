import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MCallout from './MCallout.vue'

describe('MCallout', () => {
  it.each([
    ['info', 'ℹ'],
    ['warn', '▲'],
    ['success', '✓'],
    ['danger', '✕'],
    ['note', '◈'],
  ])('renders variant %s with its glyph and class', (variant, glyph) => {
    const w = mount(MCallout, { props: { variant, content: 'body' } })
    expect(w.classes()).toContain(`callout-${variant}`)
    expect(w.text()).toContain(glyph)
  })

  it('defaults to note, renders optional title and inline md content', () => {
    const w = mount(MCallout, { props: { title: 'Locked', content: 'the `sdk` is in' } })
    expect(w.classes()).toContain('callout-note')
    expect(w.text()).toContain('Locked')
    expect(w.html()).toContain('<code>sdk</code>')
  })

  it('splits content into paragraphs on blank lines', () => {
    const w = mount(MCallout, { props: { content: 'first para\n\nsecond para' } })
    const paras = w.findAll('.mn-prose p')
    expect(paras).toHaveLength(2)
    expect(paras[0]!.text()).toBe('first para')
    expect(paras[1]!.text()).toBe('second para')
  })
})
