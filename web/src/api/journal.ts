import { apiGet, buildQuery } from './client'

// Mirrors internal/models.JournalEntry. project is absent for a global entry.
export interface JournalEntry {
  id: string
  project?: string
  session_ref: string
  summary: string
  accomplished: string[]
  deferred: string[]
  created_at: string
  updated_at: string
}

export interface JournalFilter {
  project?: string
  since?: string
}

export function listJournal(f: JournalFilter = {}): Promise<JournalEntry[]> {
  return apiGet<{ items: JournalEntry[] }>(
    `/api/v1/journal${buildQuery({ project: f.project, since: f.since })}`,
  ).then((r) => r.items)
}
