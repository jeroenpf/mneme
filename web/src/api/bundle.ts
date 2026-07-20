import { apiGet, buildQuery } from './client'

// Mirrors internal/bundle.Bundle. Returned as a single object (not wrapped
// in { items }), so apiGet<Bundle> yields it directly. The structured fields
// power the project home; BundleView reads only `markdown`. Fields the home
// consumes are typed optional because Go marshals empty slices/maps as null.
export interface PlanSummary {
  id: string
  title: string
  status: string
  active_phase?: string
  phase_current?: number
  phase_total?: number
}

export interface NextTask {
  id: string
  title: string
  phase?: string
}

export interface Blocker {
  id: string
  title: string
}

export interface PlanStats {
  total: number
  in_progress: number
  todo: number
  complete: number
}

export interface BundleDecision {
  id: string
  public_id: string
  title: string
  status: string
  decision?: string
  rationale?: string
  project?: string | null
}

export interface BundleJournal {
  id: string
  public_id: string
  session_ref: string
  summary: string
  accomplished?: string[]
  deferred?: string[]
}

export interface Bundle {
  project: string
  area?: string
  memory: Record<string, string>
  active_plan: PlanSummary | null
  plan_stats?: PlanStats
  next_tasks?: NextTask[]
  blockers?: Blocker[]
  deferred?: string[]
  decisions: BundleDecision[]
  snippets: { id: string; title: string; language: string }[]
  journal: BundleJournal[]
  markdown: string
  token_budget?: number
  estimated_tokens?: number
  truncated?: boolean
}

export function getBundle(project: string, area?: string): Promise<Bundle> {
  return apiGet<Bundle>(`/api/v1/bundle${buildQuery({ project, area })}`)
}
