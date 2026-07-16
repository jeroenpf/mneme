import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SectionNav from './SectionNav.vue'

const items = [
  { id: 'overview', title: 'Overview', kind: 'section' as const, num: '01' },
  { id: 'sp-1-7', title: 'Viewer', kind: 'subphase' as const, num: '02' },
]

// jsdom has no IntersectionObserver — install a controllable fake.
type IOCallback = (entries: Array<{ target: Element; isIntersecting: boolean }>) => void
let ioCallback: IOCallback | undefined
const observed: Element[] = []

class FakeIO {
  constructor(cb: IOCallback) {
    ioCallback = cb
  }
  observe(el: Element) {
    observed.push(el)
  }
  disconnect() {}
}

beforeEach(() => {
  observed.length = 0
  ioCallback = undefined
  vi.stubGlobal('IntersectionObserver', FakeIO)
})
afterEach(() => vi.unstubAllGlobals())

describe('SectionNav', () => {
  it('renders hash links with sequential nums and titles', () => {
    const w = mount(SectionNav, { props: { items } })
    const links = w.findAll('a')
    expect(links.map((a) => a.attributes('href'))).toEqual(['#overview', '#sp-1-7'])
    expect(links[0].text()).toContain('01')
    expect(links[0].text()).toContain('Overview')
    expect(links[1].text()).toContain('02')
  })

  it('observes the anchored elements and highlights the intersecting one', async () => {
    const overview = document.createElement('div')
    overview.id = 'overview'
    document.body.appendChild(overview)
    const w = mount(SectionNav, { props: { items }, attachTo: document.body })
    expect(observed).toContain(overview)
    ioCallback?.([{ target: overview, isIntersecting: true }])
    await w.vm.$nextTick()
    expect(w.find('a[href="#overview"]').classes()).toContain('active')
    w.unmount()
    overview.remove()
  })
})
