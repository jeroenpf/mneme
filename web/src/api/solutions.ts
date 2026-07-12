import { apiGet, buildQuery } from './client'

// Mirrors internal/models.Solution. project is absent for a global gotcha.
export interface Solution {
  id: string
  project?: string
  error_description: string
  solution: string
  tags: string[]
  source_url: string
  created_at: string
  updated_at: string
}

export interface SolutionFilter {
  project?: string
  tag?: string
}

export function listSolutions(f: SolutionFilter = {}): Promise<Solution[]> {
  return apiGet<{ items: Solution[] }>(
    `/api/v1/solutions${buildQuery({ project: f.project, tag: f.tag })}`,
  ).then((r) => r.items)
}
