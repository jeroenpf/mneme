import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import type { EnvEntry } from '@/api/env'
import { listEnv } from '@/api/env'
import { useEnv } from './useEnv'

vi.mock('@/api/env', () => ({ listEnv: vi.fn() }))

const entry = (over: Partial<EnvEntry> & Pick<EnvEntry, 'key' | 'value'>): EnvEntry => ({
  id: over.key,
  project: over.project ?? 'apollo',
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

beforeEach(() => {
  vi.mocked(listEnv).mockReset().mockResolvedValue([entry({ key: 'API_PORT', value: '8443' })])
})

describe('useEnv', () => {
  it('loads the initial project immediately', async () => {
    const { items } = useEnv(ref('apollo'))
    await flushPromises()
    expect(vi.mocked(listEnv)).toHaveBeenCalledWith('apollo')
    expect(items.value.map((e) => e.key)).toEqual(['API_PORT'])
  })

  it('refetches when the project changes', async () => {
    const project = ref('apollo')
    const { items } = useEnv(project)
    await flushPromises()
    vi.mocked(listEnv).mockResolvedValueOnce([entry({ key: 'PORT', value: '9000', project: 'hermes' })])
    project.value = 'hermes'
    await flushPromises()
    expect(vi.mocked(listEnv)).toHaveBeenLastCalledWith('hermes')
    expect(items.value.map((e) => e.value)).toEqual(['9000'])
  })

  it('clears items and skips the call for an empty project', async () => {
    const { items } = useEnv(ref(''))
    await flushPromises()
    expect(items.value).toEqual([])
    expect(vi.mocked(listEnv)).not.toHaveBeenCalled()
  })

  it('silent refresh swaps items without ever toggling loading', async () => {
    const { items, loading, refresh } = useEnv(ref('apollo'))
    await flushPromises()
    expect(loading.value).toBe(false)

    vi.mocked(listEnv).mockResolvedValueOnce([
      entry({ key: 'API_PORT', value: '8443' }),
      entry({ key: 'DB_SERVICE', value: 'postgres' }),
    ])
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → rows stay mounted
    await p
    expect(items.value.map((e) => e.key)).toEqual(['API_PORT', 'DB_SERVICE'])
    expect(loading.value).toBe(false)
  })
})
