import { apiGet, buildQuery } from './client'

// Mirrors internal/models.SearchHit. The REST endpoint wraps results in
// { items }, so unwrap like the other list endpoints.
export interface SearchHit {
  type: string
  id: string
  // Prefixed public id (doc_/dec_/snip_/sol_/jrnl_) — the stable, copyable
  // reference used for deep-linking and RefChip. Empty for memory (keyed by
  // its title) and any type without one.
  public_id?: string
  title: string
  excerpt: string
  project?: string
  score: number
  updated_at: string
}

interface SearchResponse {
  items: SearchHit[]
}

export function search(
  q: string,
  opts: { types?: string[]; project?: string; limit?: number } = {},
): Promise<SearchHit[]> {
  const query = buildQuery({
    q,
    types: opts.types,
    project: opts.project,
    limit: opts.limit,
  })
  return apiGet<SearchResponse>(`/api/v1/search${query}`).then((r) => r.items)
}

// SearchStatus mirrors GET /api/v1/search/status — embedding coverage per
// type + whether embedding is enabled (a Voyage key is configured).
export interface SearchStatus {
  enabled: boolean
  items: { type: string; embedded: number; total: number }[]
}

export function searchStatus(): Promise<SearchStatus> {
  return apiGet<SearchStatus>('/api/v1/search/status')
}
