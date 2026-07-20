import { apiGet, apiPost, buildQuery } from './client'
import type { Document, DocumentFilter, DocumentListResponse } from '@/types'

export function listDocuments(filter: DocumentFilter = {}): Promise<DocumentListResponse> {
  return apiGet<DocumentListResponse>(`/api/v1/documents${buildQuery({ ...filter })}`)
}

export function getDocument(id: string): Promise<Document> {
  return apiGet<Document>(`/api/v1/documents/${encodeURIComponent(id)}`)
}

// A compact, body-stripped revision summary from GET /documents/{id}/revisions.
export interface DocRevision {
  revision: number
  op: string
  actor: string
  target_ids: string[] | null
  title: string
  status: string
  created_at: string
}

export function listRevisions(id: string): Promise<DocRevision[]> {
  return apiGet<{ items: DocRevision[] }>(
    `/api/v1/documents/${encodeURIComponent(id)}/revisions`,
  ).then((r) => r.items)
}

export interface RestoreResult {
  restored_from: number
  new_revision: number
  doc: Document
}

// Rewinds a document to a past revision, writing a new forward revision.
export function restoreRevision(id: string, revision: number): Promise<RestoreResult> {
  return apiPost<RestoreResult>(`/api/v1/documents/${encodeURIComponent(id)}/restore`, { revision })
}
