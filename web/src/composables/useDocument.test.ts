import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import * as documentsApi from '@/api/documents'
import type { Document } from '@/types'
import { useDocument } from './useDocument'

const doc: Document = {
  id: 'a',
  title: 'A',
  type: 'plan',
  status: 'todo',
  tags: [],
  meta: {},
  body: {},
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
}

describe('useDocument', () => {
  it('fetches on creation and refetches when the id changes', async () => {
    const spy = vi.spyOn(documentsApi, 'getDocument').mockResolvedValue(doc)
    const id = ref('a')
    const { doc: result, loading } = useDocument(id)
    expect(loading.value).toBe(true)
    await vi.waitFor(() => expect(loading.value).toBe(false))
    expect(result.value?.id).toBe('a')
    id.value = 'b'
    await vi.waitFor(() => expect(spy).toHaveBeenCalledTimes(2))
    expect(spy).toHaveBeenLastCalledWith('b')
  })

  it('exposes the error and clears the stale doc', async () => {
    vi.spyOn(documentsApi, 'getDocument').mockRejectedValueOnce(new Error('404'))
    const { doc: result, error, loading } = useDocument(ref('nope'))
    await vi.waitFor(() => expect(loading.value).toBe(false))
    expect(error.value?.message).toBe('404')
    expect(result.value).toBeNull()
  })

  it('silent refresh swaps the doc without ever toggling loading', async () => {
    const spy = vi.spyOn(documentsApi, 'getDocument').mockResolvedValue(doc)
    const { doc: result, loading, refresh } = useDocument(ref('a'))
    await vi.waitFor(() => expect(loading.value).toBe(false))

    spy.mockResolvedValue({ ...doc, title: 'A updated' })
    const p = refresh({ silent: true })
    expect(loading.value).toBe(false) // no loading flip → content stays mounted
    await p
    expect(result.value?.title).toBe('A updated')
    expect(loading.value).toBe(false)
  })

  it('silent refresh keeps the current doc when the fetch fails', async () => {
    const spy = vi.spyOn(documentsApi, 'getDocument').mockResolvedValue(doc)
    const { doc: result, error, refresh } = useDocument(ref('a'))
    await vi.waitFor(() => expect(result.value?.id).toBe('a'))

    spy.mockRejectedValueOnce(new Error('boom'))
    await refresh({ silent: true })
    expect(result.value?.id).toBe('a') // stale doc retained, not blanked
    expect(error.value).toBeNull() // best-effort refetch surfaces no error
  })
})
