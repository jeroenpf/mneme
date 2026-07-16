import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import { defineComponent, h, ref } from 'vue'
import App from './App.vue'
import { useTheme } from '@/composables/useTheme'

// AppShell (in the rail) opens the shared live stream in setup; stub it since
// jsdom has no EventSource. Its behaviour is covered by useEventStream.test.ts.
vi.mock('@/composables/useEventStream', () => ({
  useEventStream: () => ({ status: ref('open'), subscribe: () => () => {}, onReconnect: () => () => {} }),
}))

const Home = defineComponent({ render: () => h('div', { 'data-test': 'page-marker' }, 'home') })
const Blank = defineComponent({ render: () => h('div') })

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Home },
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

async function mountApp() {
  const router = makeRouter()
  await router.push('/')
  await router.isReady()
  return mount(App, { global: { plugins: [router] } })
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
  useTheme().setTheme('paper')
  localStorage.clear()
})

describe('App', () => {
  it('renders the persistent rail around the routed view', async () => {
    const w = await mountApp()
    expect(w.find('[data-test="to-memory"]').exists()).toBe(true)
    expect(w.find('[data-test="page-marker"]').text()).toBe('home')
  })

  it('no longer renders the floating ThemePicker overlay (it lives in the rail)', async () => {
    const w = await mountApp()
    expect(w.find('.theme-picker-float').exists()).toBe(false)
    // ThemePicker is mounted exactly once — in the rail.
    expect(w.findAll('button[data-test^="theme-"]')).toHaveLength(3)
  })
})
