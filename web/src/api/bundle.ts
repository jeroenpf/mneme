import { apiGet, buildQuery } from './client'

// Mirrors internal/bundle.Bundle. Returned as a single object (not wrapped
// in { items }), so apiGet<Bundle> yields it directly.
export interface PlanSummary {
  id: string
  title: string
  status: string
  phase_current?: number
  phase_total?: number
}

export interface Bundle {
  project: string
  area?: string
  memory: Record<string, string>
  active_plan: PlanSummary | null
  decisions: { id: string; title: string; status: string }[]
  snippets: { id: string; title: string; language: string }[]
  journal: { id: string; session_ref: string; summary: string }[]
  markdown: string
}

export function getBundle(project: string, area?: string): Promise<Bundle> {
  return apiGet<Bundle>(`/api/v1/bundle${buildQuery({ project, area })}`)
}
