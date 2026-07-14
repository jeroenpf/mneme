import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { EnvEntry } from '@/api/env'
import { deleteEnv, listEnv, setEnv } from '@/api/env'
import { listProjects } from '@/api/projects'
import EnvView from './EnvView.vue'

vi.mock('@/api/env', () => ({
  listEnv: vi.fn(),
  setEnv: vi.fn(),
  deleteEnv: vi.fn(),
}))
vi.mock('@/api/projects', () => ({ listProjects: vi.fn() }))

const portEntry: EnvEntry = {
  id: 'e1',
  project: 'apollo',
  key: 'API_PORT',
  value: '8443',
  description: 'https port',
  updated_at: '2026-07-01T00:00:00Z',
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ render: () => h('div') }) },
      { path: '/env', component: EnvView },
    ],
  })
}

async function mountView() {
  const router = makeRouter()
  await router.push('/env')
  await router.isReady()
  const w = mount(EnvView, { global: { plugins: [router] } })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(listProjects)
    .mockReset()
    .mockResolvedValue([
      {
        id: 'p1',
        name: 'apollo',
        slug: 'apollo',
        created_at: '',
        counts: { todo: 0, 'in-progress': 0, complete: 0, blocked: 0, archived: 0, total: 0 },
      },
    ])
  vi.mocked(listEnv).mockReset().mockResolvedValue([portEntry])
  vi.mocked(setEnv).mockReset().mockResolvedValue(portEntry)
  vi.mocked(deleteEnv).mockReset().mockResolvedValue(undefined)
})

describe('EnvView', () => {
  it('renders no in-page topbar — the rail owns navigation', async () => {
    const w = await mountView()
    expect(w.find('header.topbar').exists()).toBe(false)
  })

  it('always shows the never-store-secrets warning', async () => {
    const w = await mountView()
    expect(w.get('[data-test="secrets-warning"]').text().toLowerCase()).toContain('never store secrets')
  })

  it('renders a project entry key, value and description', async () => {
    const w = await mountView()
    const row = w.get('[data-test="entry"]')
    expect(row.text()).toContain('API_PORT')
    expect((row.get('input.entry-value').element as HTMLInputElement).value).toBe('8443')
    expect((row.get('input.entry-desc').element as HTMLInputElement).value).toBe('https port')
  })

  it('saves an edited value on blur, preserving the description', async () => {
    const w = await mountView()
    const input = w.get('[data-test="entry"] input.entry-value')
    await input.setValue('9000')
    await input.trigger('blur')
    await flushPromises()
    expect(vi.mocked(setEnv)).toHaveBeenCalledWith(
      expect.objectContaining({ project: 'apollo', key: 'API_PORT', value: '9000', description: 'https port' }),
    )
  })

  it('deletes an entry via the delete control', async () => {
    const w = await mountView()
    await w.get('[data-test="entry-delete"]').trigger('click')
    await flushPromises()
    expect(vi.mocked(deleteEnv)).toHaveBeenCalledWith(
      expect.objectContaining({ project: 'apollo', key: 'API_PORT' }),
    )
  })

  it('adds a new entry through the add form', async () => {
    const w = await mountView()
    const form = w.get('[data-test="add-row"]')
    await form.get('input.add-key').setValue('DB_SERVICE')
    await form.get('input.add-value').setValue('postgres')
    await form.get('[data-test="add-submit"]').trigger('click')
    await flushPromises()
    expect(vi.mocked(setEnv)).toHaveBeenCalledWith(
      expect.objectContaining({ project: 'apollo', key: 'DB_SERVICE', value: 'postgres' }),
    )
  })
})
