import { describe, expect, it } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import Topbar from './Topbar.vue'

const global = { stubs: { RouterLink: RouterLinkStub } }

describe('Topbar', () => {
  it('emits the search text as its v-model', async () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global })
    await w.find('input[type="search"]').setValue('zigbee')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['zigbee'])
  })

  it('focuses the search input when / is pressed outside a field', async () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global, attachTo: document.body })
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }))
    expect(document.activeElement).toBe(w.find('input[type="search"]').element)
    w.unmount()
  })

  it('links to the memory page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global })
    const link = w.findComponent(RouterLinkStub)
    expect(link.props('to')).toBe('/memory')
    expect(link.attributes('data-test')).toBe('to-memory')
  })

  it('links to the decisions page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global })
    const decisions = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-decisions')
    expect(decisions).toBeTruthy()
    expect(decisions!.props('to')).toBe('/decisions')
  })

  it('links to the snippets page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global })
    const snippets = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-snippets')
    expect(snippets).toBeTruthy()
    expect(snippets!.props('to')).toBe('/snippets')
  })

  it('links to the journal page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global })
    const journal = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-journal')
    expect(journal).toBeTruthy()
    expect(journal!.props('to')).toBe('/journal')
  })
})
