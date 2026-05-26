import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiGet, buildQuery } from './client'

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
