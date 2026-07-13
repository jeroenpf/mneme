import { apiGet, buildQuery } from './client'

// Mirrors internal/models.SearchHit. The REST endpoint wraps results in
// { items }, so unwrap like the other list endpoints.
export interface SearchHit {
  type: string
  id: string
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
