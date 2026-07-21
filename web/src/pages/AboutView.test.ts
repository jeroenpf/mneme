import { describe, expect, it } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import AboutView from './AboutView.vue'

const mountView = () =>
  mount(AboutView, { global: { stubs: { RouterLink: RouterLinkStub } } })

describe('AboutView', () => {
  it('renders every section', () => {
    const w = mountView()
    for (const s of ['what', 'why', 'idea', 'not', 'delineation']) {
      expect(w.find(`[data-test="${s}"]`).exists(), `section ${s}`).toBe(true)
    }
  })

  it('describes what mneme is and is not', () => {
    const text = mountView().text()
    expect(text).toContain('local, single-user knowledge service')
    expect(text).toContain('Not a team wiki')
    expect(text).toContain('Not an issue tracker')
  })

  it('states the delineation rule and the split', () => {
    const w = mountView()
    const sec = w.get('[data-test="delineation"]').text()
    expect(sec).toContain('born in mneme')
    expect(sec).toContain('never copies')
    // The table pairs each document class with its home.
    const rows = w.get('[data-test="delineation"]').findAll('tbody tr')
    expect(rows.length).toBeGreaterThanOrEqual(4)
    expect(sec).toContain('Active plans')
    expect(sec).toContain('Accepted specs')
  })

  it('links onward to the Help page', () => {
    const w = mountView()
    const links = w.findAllComponents(RouterLinkStub).map((l) => l.props('to'))
    expect(links).toContain('/help')
  })
})
