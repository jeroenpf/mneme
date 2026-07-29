import { apiGet } from './client'

// One related entity, enriched server-side: the endpoint's public id (or raw
// slug when dangling), its kind, a display title, and the edge's rel type and
// direction relative to the queried document.
export interface RelatedEntry {
  id: string
  kind: 'document' | 'decision' | 'snippet' | 'solution' | 'journal'
  title: string
  rel_type: string
  direction: 'out' | 'in'
  doc_status?: string
  dangling?: boolean
}

export interface RelatedBundle {
  links: RelatedEntry[]
  mentions: RelatedEntry[]
  mentioned_by: RelatedEntry[]
}

export function getRelated(id: string): Promise<RelatedBundle> {
  return apiGet<RelatedBundle>(`/api/v1/documents/${encodeURIComponent(id)}/related`)
}
