import { describe, expect, it } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'
import Topbar from './Topbar.vue'

// Topbar now calls useRouter() (Enter → /search), so every mount needs a
// router plugin — without it useRouter() returns undefined and onEnter
// throws. RouterLinkStub is kept so the link assertions stay simple.
function makeGlobal() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/search', component: defineComponent({ render: () => h('div') }) },
    ],
  })
  return { plugins: [router], stubs: { RouterLink: RouterLinkStub } }
}

describe('Topbar', () => {
  it('emits the search text as its v-model', async () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    await w.find('input[type="search"]').setValue('zigbee')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['zigbee'])
  })

  it('focuses the search input when / is pressed outside a field', async () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal(), attachTo: document.body })
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }))
    expect(document.activeElement).toBe(w.find('input[type="search"]').element)
    w.unmount()
  })

  it('links to the memory page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const link = w.findComponent(RouterLinkStub)
    expect(link.props('to')).toBe('/memory')
    expect(link.attributes('data-test')).toBe('to-memory')
  })

  it('links to the decisions page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const decisions = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-decisions')
    expect(decisions).toBeTruthy()
    expect(decisions!.props('to')).toBe('/decisions')
  })

  it('links to the snippets page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const snippets = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-snippets')
    expect(snippets).toBeTruthy()
    expect(snippets!.props('to')).toBe('/snippets')
  })

  it('links to the journal page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const journal = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-journal')
    expect(journal).toBeTruthy()
    expect(journal!.props('to')).toBe('/journal')
  })

  it('links to the solutions page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const solutions = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-solutions')
    expect(solutions).toBeTruthy()
    expect(solutions!.props('to')).toBe('/solutions')
  })

  it('links to the bundle page', () => {
    const w = mount(Topbar, { props: { modelValue: '' }, global: makeGlobal() })
    const bundle = w
      .findAllComponents(RouterLinkStub)
      .find((l) => l.attributes('data-test') === 'to-bundle')
    expect(bundle).toBeTruthy()
    expect(bundle!.props('to')).toBe('/bundle')
  })

  it('navigates to /search on enter', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: defineComponent({ render: () => h('div') }) },
        { path: '/search', component: defineComponent({ render: () => h('div') }) },
      ],
    })
    const w = mount(Topbar, { props: { modelValue: '' }, global: { plugins: [router] } })
    await w.find('input[type="search"]').setValue('zigbee')
    await w.find('input[type="search"]').trigger('keyup.enter')
    await router.isReady()
    expect(router.currentRoute.value.fullPath).toBe('/search?q=zigbee')
  })
})
