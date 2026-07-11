import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MText from './MText.vue'

describe('MText', () => {
  it('renders inline markdown in content', () => {
    const w = mount(MText, { props: { content: 'a **bold** `code` move' } })
    expect(w.find('p').html()).toContain('<strong>bold</strong>')
    expect(w.find('p').html()).toContain('<code>code</code>')
  })

  it('renders nothing for empty content', () => {
    expect(mount(MText, { props: {} }).find('p').exists()).toBe(false)
  })
})
