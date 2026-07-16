import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import * as documentsApi from '@/api/documents'
import { useDocuments } from './useDocuments'

describe('useDocuments', () => {
  it('fetches on mount and exposes items', async () => {
    const spy = vi.spyOn(documentsApi, 'listDocuments').mockResolvedValue({
      items: [
        {
          id: 'vehicle-api',
          title: 'Vehicle API',
          type: 'plan',
          status: 'todo',
          tags: [],
          meta: {},
          body: {},
          created_at: '',
          updated_at: '',
        },
      ],
      next_cursor: null,
    })

    const filter = ref({})
    const { items, loading, error } = useDocuments(filter)

    expect(loading.value).toBe(true)
    await vi.waitFor(() => expect(loading.value).toBe(false))

    expect(items.value).toHaveLength(1)
    expect(items.value[0].id).toBe('vehicle-api')
    expect(error.value).toBeNull()
    expect(spy).toHaveBeenCalledOnce()
  })

  it('refetches when the filter ref changes', async () => {
    const spy = vi.spyOn(documentsApi, 'listDocuments').mockResolvedValue({
      items: [],
      next_cursor: null,
    })

    const filter = ref<{ project?: string }>({})
    useDocuments(filter)

    await vi.waitFor(() => expect(spy).toHaveBeenCalledOnce())

    filter.value = { project: 'apollo' }
    await nextTick()
    await vi.waitFor(() => expect(spy).toHaveBeenCalledTimes(2))

    expect(spy.mock.calls[1][0]).toEqual({ project: 'apollo' })
  })

  it('exposes error when the api throws', async () => {
    vi.spyOn(documentsApi, 'listDocuments').mockRejectedValueOnce(new Error('boom'))
    const { error, loading } = useDocuments(ref({}))
    await vi.waitFor(() => expect(loading.value).toBe(false))
    expect(error.value?.message).toBe('boom')
  })

  it('silent refresh swaps items without ever toggling loading', async () => {
    const doc = (id: string) => ({
      id,
      title: id,
      type: 'plan' as const,
      status: 'todo' as const,
      tags: [],
      meta: {},
      body: {},
      created_at: '',
      updated_at: '',
    })
    const spy = vi
      .spyOn(documentsApi, 'listDocuments')
      .mockResolvedValue({ items: [doc('a')], next_cursor: null })
    const { items, loading, refresh } = useDocuments(ref({}))
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue({ items: [doc('a'), doc('b')], next_cursor: null })
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → grid stays mounted
    await p
    expect(items.value).toHaveLength(2)
    expect(loading.value).toBe(false)
  })
})
