import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { MemoryEntry } from '@/api/memory'
import { deleteMemory, listMemory, setMemory } from '@/api/memory'
import { useLiveRefresh } from '@/composables/useLiveRefresh'
import MemoryView from './MemoryView.vue'

vi.mock('@/api/memory', () => ({
  listMemory: vi.fn(),
  setMemory: vi.fn(),
  deleteMemory: vi.fn(),
}))
// Stub the live layer: it constructs a real EventSource (absent in jsdom).
// Its own behaviour is covered by useLiveRefresh.test.ts.
vi.mock('@/composables/useLiveRefresh', () => ({ useLiveRefresh: vi.fn() }))

const globalEntry: MemoryEntry = {
  id: 'g1',
  scope: 'global',
  key: 'editor',
  value: 'neovim',
  updated_at: '2026-07-01T00:00:00Z',
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/memory', component: MemoryView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/memory')
  await router.isReady()
  const w = mount(MemoryView, { global: { plugins: [router] } })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listMemory).mockReset().mockResolvedValue([globalEntry])
  vi.mocked(setMemory).mockReset().mockResolvedValue(globalEntry)
  vi.mocked(deleteMemory).mockReset().mockResolvedValue(undefined)
})

describe('MemoryView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountView()
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('renders a global entry key and value', async () => {
    const w = await mountView()
    const row = w.get('[data-test="entry"]')
    expect(row.text()).toContain('editor')
    expect((row.get('input.entry-value').element as HTMLInputElement).value).toBe('neovim')
  })

  it('saves an edited value on blur', async () => {
    const w = await mountView()
    const input = w.get('[data-test="entry"] input.entry-value')
    await input.setValue('goland')
    await input.trigger('blur')
    await flushPromises()
    expect(vi.mocked(setMemory)).toHaveBeenCalledWith(
      expect.objectContaining({ scope: 'global', key: 'editor', value: 'goland' }),
    )
  })

  it('does not save when the value is unchanged', async () => {
    const w = await mountView()
    const input = w.get('[data-test="entry"] input.entry-value')
    await input.trigger('blur')
    await flushPromises()
    expect(vi.mocked(setMemory)).not.toHaveBeenCalled()
  })

  it('deletes an entry via the delete control', async () => {
    const w = await mountView()
    await w.get('[data-test="entry-delete"]').trigger('click')
    await flushPromises()
    expect(vi.mocked(deleteMemory)).toHaveBeenCalledWith(
      expect.objectContaining({ scope: 'global', key: 'editor' }),
    )
  })

  it('adds a new global entry through the add form', async () => {
    const w = await mountView()
    const form = w.get('[data-test="add-global"]')
    await form.get('input.add-key').setValue('shell')
    await form.get('input.add-value').setValue('zsh')
    await form.get('[data-test="add-submit"]').trigger('click')
    await flushPromises()
    expect(vi.mocked(setMemory)).toHaveBeenCalledWith(
      expect.objectContaining({ scope: 'global', key: 'shell', value: 'zsh' }),
    )
  })

  it('subscribes to live memory events and targets the changed entry by key', async () => {
    await mountView()
    expect(useLiveRefresh).toHaveBeenCalledWith('memory', expect.anything())
    const opts = vi.mocked(useLiveRefresh).mock.calls[0]![1]
    expect(opts.flashTarget?.({ type: 'memory', id: 'editor' })).toBe('[data-flash-id="editor"]')
  })
})
