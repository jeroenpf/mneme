const BASE = import.meta.env.VITE_API_URL ?? ''

export class ApiError extends Error {
  override name = 'ApiError'
  public status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

type QueryValue = string | number | boolean | string[] | undefined

export function buildQuery(params: Record<string, QueryValue>): string {
  const search = new URLSearchParams()
  for (const [key, raw] of Object.entries(params)) {
    if (raw === undefined || raw === '') continue
    if (Array.isArray(raw)) {
      if (raw.length === 0) continue
      search.set(key, raw.join(','))
    } else {
      search.set(key, String(raw))
    }
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}

export async function apiGet<T>(path: string): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${BASE}${path}`, {
      headers: { Accept: 'application/json' },
    })
  } catch (err) {
    throw new ApiError(0, err instanceof Error ? err.message : 'network error')
  }
  if (!res.ok) {
    let msg = `request failed: ${res.status}`
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) msg = body.error
    } catch {
      // body wasn't JSON; keep the status-based message
    }
    throw new ApiError(res.status, msg)
  }
  return (await res.json()) as T
}
