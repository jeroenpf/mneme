import { apiGet, buildQuery } from './client'

// Mirrors internal/models.Snippet. project is absent for a global snippet.
export interface Snippet {
  id: string
  public_id?: string
  title: string
  project?: string
  language: string
  content: string
  tags: string[]
  description: string
  created_at: string
  updated_at: string
}

export interface SnippetFilter {
  project?: string
  language?: string
  tag?: string
}

export function listSnippets(f: SnippetFilter = {}): Promise<Snippet[]> {
  return apiGet<{ items: Snippet[] }>(
    `/api/v1/snippets${buildQuery({ project: f.project, language: f.language, tag: f.tag })}`,
  ).then((r) => r.items)
}
