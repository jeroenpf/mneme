import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiDelete, apiGet, apiPut, buildQuery } from './client'

describe('buildQuery', () => {
  it('returns empty string when no params', () => {
    expect(buildQuery({})).toBe('')
  })

  it('skips undefined and empty values', () => {
    expect(buildQuery({ a: 'x', b: undefined, c: '' })).toBe('?a=x')
  })

  it('joins array values with commas', () => {
    expect(buildQuery({ tags: ['go', 'api'] })).toBe('?tags=go%2Capi')
  })

  it('encodes special chars', () => {
    expect(buildQuery({ q: 'vehicle OR pricing' })).toBe('?q=vehicle+OR+pricing')
  })
})

describe('apiGet', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('returns parsed JSON for 200 responses', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiGet<{ ok: boolean }>('/x')).resolves.toEqual({ ok: true })
  })

  it('throws ApiError with status and server message on 4xx', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'not found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiGet('/x')).rejects.toMatchObject({
      name: 'ApiError',
      status: 404,
      message: 'not found',
    })
  })

  it('throws ApiError for transport failures', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new TypeError('boom'))
    await expect(apiGet('/x')).rejects.toBeInstanceOf(ApiError)
  })
})

describe('apiPut', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })
  afterEach(() => {
    globalThis.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('sends the JSON body and returns the parsed entry', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify({ id: '1', value: 'neovim' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiPut('/api/v1/memory/global/editor', { value: 'neovim' })).resolves.toEqual({
      id: '1',
      value: 'neovim',
    })
    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/memory/global/editor',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ value: 'neovim' }),
      }),
    )
  })

  it('throws ApiError with the server message on 4xx', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'unknown project' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiPut('/x', { value: 'v' })).rejects.toMatchObject({
      name: 'ApiError',
      status: 400,
      message: 'unknown project',
    })
  })
})

describe('apiDelete', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })
  afterEach(() => {
    globalThis.fetch = originalFetch
    vi.restoreAllMocks()
  })

  it('resolves on 204 No Content (empty body)', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(null, { status: 204 }),
    )
    await expect(apiDelete('/api/v1/memory/global/k')).resolves.toBeUndefined()
    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/memory/global/k',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws ApiError on non-2xx', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'not found' }), {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiDelete('/x')).rejects.toBeInstanceOf(ApiError)
  })
})
