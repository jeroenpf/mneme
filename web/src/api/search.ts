import { apiGet, apiPost, buildQuery } from './client'

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

// SearchStatus mirrors GET /api/v1/search/status — per-type reconciliation
// buckets, provider identity, live queue depth, and the last reconciliation
// time. `enabled` is true when an embedding provider (a Voyage key) is
// configured; when false the store falls back to lexical-only search.
export interface EmbeddingTypeStatus {
  type: string
  total: number
  embedded: number
  reconciled: number
  missing: number
  stale: number
  orphaned: number
  failed: number
}

export interface EmbeddingProvider {
  name: string
  model: string
  enabled: boolean
}

export interface SearchStatus {
  enabled: boolean
  provider: EmbeddingProvider
  items: EmbeddingTypeStatus[]
  queue_depth: number
  last_reconcile?: string
}

export function searchStatus(): Promise<SearchStatus> {
  return apiGet<SearchStatus>('/api/v1/search/status')
}

// reindexFailed re-enqueues every terminally-failed source for another
// embedding attempt (POST /api/v1/search/reindex-failed), returning the count.
export function reindexFailed(): Promise<{ retried: number }> {
  return apiPost<{ retried: number }>('/api/v1/search/reindex-failed')
}
