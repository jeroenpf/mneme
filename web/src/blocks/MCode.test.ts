import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import MCode from './MCode.vue'

describe('MCode', () => {
  it('renders lang badge, filename, and highlighted content', async () => {
    const w = mount(MCode, {
      props: { lang: 'go', filename: 'main.go', content: 'func main() {}' },
    })
    await flushPromises()
    expect(w.find('[data-test="lang"]').text()).toBe('go')
    expect(w.find('[data-test="filename"]').text()).toBe('main.go')
    // The lazy prism import spans real module-load ticks, not just one
    // microtask flush — poll until the highlight lands.
    await vi.waitFor(() => expect(w.find('code').html()).toContain('token'))
  })

  it('renders escaped text when lang is unknown', async () => {
    const w = mount(MCode, { props: { content: 'a < b' } })
    await flushPromises()
    expect(w.find('code').text()).toContain('a < b')
    expect(w.find('[data-test="lang"]').exists()).toBe(false)
  })

  it('copies the raw content to the clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    const w = mount(MCode, { props: { lang: 'go', content: 'func main() {}' } })
    await w.find('[data-test="copy"]').trigger('click')
    expect(writeText).toHaveBeenCalledWith('func main() {}')
    vi.unstubAllGlobals()
  })
})
