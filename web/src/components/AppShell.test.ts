import { describe, it, expect, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { defineComponent, h } from 'vue'
import AppShell from './AppShell.vue'
import { useTheme } from '@/composables/useTheme'

const Blank = defineComponent({ render: () => h('div') })

// The rail links point at the eight primary destinations (plus /search for the
// search box), so the router needs a record for each — real RouterLink resolves
// an href and the active class only against registered routes.
function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Blank },
      { path: '/memory', component: Blank },
      { path: '/decisions', component: Blank },
      { path: '/snippets', component: Blank },
      { path: '/journal', component: Blank },
      { path: '/solutions', component: Blank },
      { path: '/env', component: Blank },
      { path: '/bundle', component: Blank },
      { path: '/search', component: Blank },
    ],
  })
}

async function mountShell(router: Router, slot = '<p class="page">page body</p>') {
  await router.push('/')
  await router.isReady()
  return mount(AppShell, {
    global: { plugins: [router] },
    slots: { default: slot },
  })
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
  // Reset the useTheme module singleton (ThemePicker lives in the rail).
  useTheme().setTheme('paper')
  localStorage.clear()
})

// The primary destinations, in the rail's order (matches the mockup).
const NAV = [
  ['to-registry', '/'],
  ['to-memory', '/memory'],
  ['to-decisions', '/decisions'],
  ['to-snippets', '/snippets'],
  ['to-journal', '/journal'],
  ['to-solutions', '/solutions'],
  ['to-env', '/env'],
  ['to-bundle', '/bundle'],
] as const

describe('AppShell', () => {
  it('renders a rail link for each primary destination', async () => {
    const w = await mountShell(makeRouter())
    for (const [test, path] of NAV) {
      const link = w.find(`[data-test="${test}"]`)
      expect(link.exists(), `${test} link should exist`).toBe(true)
      expect(link.attributes('href')).toBe(path)
    }
  })

  it('renders the brand wordmark', async () => {
    const w = await mountShell(makeRouter())
    expect(w.text()).toContain('mneme')
  })

  it('renders default slot content in the content area', async () => {
    const w = await mountShell(makeRouter())
    expect(w.find('.page').text()).toBe('page body')
  })

  it('mounts the ThemePicker in the rail', async () => {
    const w = await mountShell(makeRouter())
    expect(w.findAll('button[data-test^="theme-"]')).toHaveLength(3)
  })

  // Active state comes from vue-router's router-link-active classes (real
  // RouterLink required — RouterLinkStub never emits them).
  it("marks the current route's rail link active", async () => {
    const w = await mountShell(makeRouter())
    await w.vm.$router.push('/memory')
    await w.vm.$nextTick()

    expect(w.find('[data-test="to-memory"]').classes()).toContain('router-link-active')
  })

  it('does not mark the root (registry) link active on a sibling route', async () => {
    const w = await mountShell(makeRouter())
    await w.vm.$router.push('/memory')
    await w.vm.$nextTick()

    // The active-bar styling keys on router-link-exact-active, so "/" must not
    // carry it on /memory — otherwise the rail would show two active items.
    const registry = w.find('[data-test="to-registry"]')
    expect(registry.classes()).not.toContain('router-link-exact-active')
    expect(registry.classes()).not.toContain('router-link-active')
  })

  it('marks the registry link active on the root route', async () => {
    const w = await mountShell(makeRouter()) // mountShell pushes '/'
    expect(w.find('[data-test="to-registry"]').classes()).toContain('router-link-exact-active')
  })

  // Global search lives in the rail (lifted from Topbar): a local ref, Enter
  // routes to /search, "/" focuses it. It is NOT wired to registry filtering.
  it('navigates to /search on Enter with the trimmed query', async () => {
    const w = await mountShell(makeRouter())
    const input = w.find('input[type="search"]')
    await input.setValue('zigbee')
    await input.trigger('keyup.enter')
    await flushPromises()
    expect(w.vm.$router.currentRoute.value.fullPath).toBe('/search?q=zigbee')
  })

  it('does not navigate on Enter when the query is blank', async () => {
    const w = await mountShell(makeRouter())
    const input = w.find('input[type="search"]')
    await input.setValue('   ')
    await input.trigger('keyup.enter')
    await flushPromises()
    expect(w.vm.$router.currentRoute.value.fullPath).toBe('/')
  })

  it('focuses the rail search when "/" is pressed outside a field', async () => {
    const router = makeRouter()
    await router.push('/')
    await router.isReady()
    const w = mount(AppShell, {
      global: { plugins: [router] },
      slots: { default: '<div />' },
      attachTo: document.body,
    })
    window.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }))
    expect(document.activeElement).toBe(w.find('input[type="search"]').element)
    w.unmount()
  })
})
