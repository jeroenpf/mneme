import { apiGet, buildQuery } from './client'
import type { Document, DocumentFilter, DocumentListResponse } from '@/types'

export function listDocuments(filter: DocumentFilter = {}): Promise<DocumentListResponse> {
  return apiGet<DocumentListResponse>(`/api/v1/documents${buildQuery({ ...filter })}`)
}

export function getDocument(id: string): Promise<Document> {
  return apiGet<Document>(`/api/v1/documents/${encodeURIComponent(id)}`)
}
