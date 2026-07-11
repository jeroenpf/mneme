import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Topbar from './Topbar.vue'

describe('Topbar', () => {
  it('emits the search text as its v-model', async () => {
    const w = mount(Topbar, { props: { modelValue: '' } })
    await w.find('input[type="search"]').setValue('zigbee')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['zigbee'])
  })

  it('focuses the search input when / is pressed outside a field', async () => {
    const w = mount(Topbar, { props: { modelValue: '' }, attachTo: document.body })
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }))
    expect(document.activeElement).toBe(w.find('input[type="search"]').element)
    w.unmount()
  })
})
