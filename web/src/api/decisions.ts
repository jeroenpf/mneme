import { apiGet, buildQuery } from './client'

export type DecisionStatus = 'proposed' | 'accepted' | 'deprecated'

// Mirrors internal/models.Decision. project is absent for a global decision.
export interface Decision {
  id: string
  title: string
  project?: string
  decision: string
  rationale: string
  alternatives: string
  consequences: string
  status: DecisionStatus
  created_at: string
  updated_at: string
}

export interface DecisionFilter {
  project?: string
  status?: DecisionStatus
}

export function listDecisions(f: DecisionFilter = {}): Promise<Decision[]> {
  return apiGet<{ items: Decision[] }>(
    `/api/v1/decisions${buildQuery({ project: f.project, status: f.status })}`,
  ).then((r) => r.items)
}
