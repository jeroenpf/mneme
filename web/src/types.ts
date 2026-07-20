// Mirrors of the Go models in internal/models/models.go.
// Update both files together when shapes change.

export type DocumentType =
  | 'plan' | 'report' | 'spec' | 'adr' | 'brainstorm' | 'journal'

export type DocumentStatus =
  | 'todo' | 'in-progress' | 'complete' | 'blocked' | 'archived'

export interface Document {
  id: string
  public_id?: string
  title: string
  project?: string
  category?: string
  type: DocumentType
  status: DocumentStatus
  ticket?: string
  repo?: string
  tags: string[]
  phase_current?: number
  phase_total?: number
  meta: Record<string, unknown>
  body: Record<string, unknown>
  revision?: number
  created_at: string
  updated_at: string
}

// A task entry inside subphase.tasks[] / task-list.tasks[]. Not a block —
// no type field; validateBody only checks shape.
export interface TaskItem {
  id: string
  title: string
  done?: boolean
  content?: string
  tags?: string[]
}

export interface ProjectCounts {
  todo: number
  'in-progress': number
  complete: number
  blocked: number
  archived: number
  total: number
}

export interface ProjectStats {
  id: string
  public_id?: string
  name: string
  slug: string
  description?: string
  created_at: string
  counts: ProjectCounts
}

export interface DocumentListResponse {
  items: Document[]
  next_cursor: string | null
}

export interface DocumentFilter {
  project?: string
  type?: DocumentType
  status?: DocumentStatus
  tags?: string[]
  q?: string
  limit?: number
  cursor?: string
}
