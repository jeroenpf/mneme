import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
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
})
