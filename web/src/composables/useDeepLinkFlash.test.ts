import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { useDeepLinkFlash } from './useDeepLinkFlash'

function host(ready: () => boolean) {
  return defineComponent({
    setup() {
      useDeepLinkFlash(ready)
      return () =>
        h('ul', [
          h('li', { 'data-flash-id': 'd1' }, 'one'),
          h('li', { 'data-flash-id': 'd2' }, 'two'),
        ])
    },
  })
}

async function mountAt(comp: ReturnType<typeof host>, path: string) {
  const router: Router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/x', component: comp }],
  })
  await router.push(path)
  await router.isReady()
  const w = mount(comp, { global: { plugins: [router] }, attachTo: document.body })
  await flushPromises()
  await nextTick()
  return w
}

beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn()
  vi.stubGlobal('matchMedia', () => ({ matches: false }))
  document.body.innerHTML = ''
})

describe('useDeepLinkFlash', () => {
  it('scrolls to and flashes the row named by ?flash', async () => {
    await mountAt(host(() => true), '/x?flash=d2')
    expect(Element.prototype.scrollIntoView).toHaveBeenCalledTimes(1)
    expect(document.querySelector('[data-flash-id="d2"]')?.classList.contains('mn-flash')).toBe(true)
    expect(document.querySelector('[data-flash-id="d1"]')?.classList.contains('mn-flash')).toBe(false)
  })

  it('also reveals a row by its public reference id (data-ref-id)', async () => {
    const comp = defineComponent({
      setup() {
        useDeepLinkFlash(() => true)
        return () => h('ul', [h('li', { 'data-ref-id': 'dec_1' }, 'x')])
      },
    })
    await mountAt(comp as ReturnType<typeof host>, '/x?flash=dec_1')
    expect(document.querySelector('[data-ref-id="dec_1"]')?.classList.contains('mn-flash')).toBe(true)
  })

  it('does nothing without a ?flash param', async () => {
    await mountAt(host(() => true), '/x')
    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled()
  })

  it('waits until the list is ready before revealing', async () => {
    const ready = ref(false)
    await mountAt(host(() => ready.value), '/x?flash=d1')
    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled()
    ready.value = true
    await flushPromises()
    await nextTick()
    expect(Element.prototype.scrollIntoView).toHaveBeenCalledTimes(1)
    expect(document.querySelector('[data-flash-id="d1"]')?.classList.contains('mn-flash')).toBe(true)
  })
})
