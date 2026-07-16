import { computed, type ComputedRef } from 'vue'
import { useRoute, useRouter, type LocationQueryValue } from 'vue-router'
import type { Document, DocumentFilter, DocumentStatus, DocumentType } from '@/types'

export type SortKey = 'updated' | 'created' | 'title'

export interface RegistryFilterState {
  statuses: DocumentStatus[]
  type?: DocumentType
  project?: string
  q?: string
  sort: SortKey
}

export interface UseRegistryFiltersResult {
  state: ComputedRef<RegistryFilterState>
  apiFilter: ComputedRef<DocumentFilter>
  update: (patch: Partial<RegistryFilterState>) => void
}

// The statuses offered as filter pills, in display/canonical order.
// `archived` is deliberately absent — it lives in its own collapsed
// section, independent of these pills.
export const PILL_STATUSES: readonly DocumentStatus[] = ['todo', 'in-progress', 'complete', 'blocked']
// Shown when no explicit selection is in the URL: the working set,
// hiding `complete` (and `archived`).
export const DEFAULT_STATUSES: readonly DocumentStatus[] = ['todo', 'in-progress', 'blocked']

const TYPES: readonly DocumentType[] = ['plan', 'report', 'spec', 'adr', 'brainstorm', 'journal']
const SORTS: readonly SortKey[] = ['updated', 'created', 'title']

function first(v: LocationQueryValue | LocationQueryValue[] | undefined): string | undefined {
  const s = Array.isArray(v) ? v[0] : v
  return typeof s === 'string' && s !== '' ? s : undefined
}

function pick<T extends string>(raw: string | undefined, allowed: readonly T[]): T | undefined {
  return allowed.includes(raw as T) ? (raw as T) : undefined
}

// Multi-select status <-> URL round-trip. Absent → the default trio;
// the `none` sentinel → an explicit empty selection (show nothing);
// a CSV → those statuses, canonicalised to PILL_STATUSES order with
// duplicates and unknown values dropped.
function parseStatuses(raw: string | undefined): DocumentStatus[] {
  if (raw === undefined) return [...DEFAULT_STATUSES]
  if (raw === 'none') return []
  const want = new Set(raw.split(','))
  return PILL_STATUSES.filter((s) => want.has(s))
}

// Inverse of parseStatuses. Returns undefined (omit the param) when the
// selection equals the default trio, `none` for an empty selection, and
// a canonical CSV otherwise.
function serializeStatuses(statuses: DocumentStatus[]): string | undefined {
  const csv = PILL_STATUSES.filter((s) => statuses.includes(s)).join(',')
  if (csv === DEFAULT_STATUSES.join(',')) return undefined
  return csv === '' ? 'none' : csv
}

// Filter state lives in the URL query (?status=&type=&project=&q=&sort=)
// so views are linkable and survive reload. sort is client-side only and
// never reaches the API.
export function useRegistryFilters(): UseRegistryFiltersResult {
  const route = useRoute()
  const router = useRouter()

  const state = computed<RegistryFilterState>(() => ({
    statuses: parseStatuses(first(route.query.status)),
    type: pick(first(route.query.type), TYPES),
    project: first(route.query.project),
    q: first(route.query.q),
    sort: pick(first(route.query.sort), SORTS) ?? 'updated',
  }))

  // Status is filtered client-side (RegistryView already fetches the page
  // and partitions it), so it never reaches the API — only project/type/q
  // do, which is what triggers a refetch.
  const apiFilter = computed<DocumentFilter>(() => {
    const s = state.value
    const f: DocumentFilter = {}
    if (s.type) f.type = s.type
    if (s.project) f.project = s.project
    if (s.q) f.q = s.q
    return f
  })

  function update(patch: Partial<RegistryFilterState>) {
    const next = { ...state.value, ...patch }
    const query: Record<string, string> = {}
    const status = serializeStatuses(next.statuses)
    if (status !== undefined) query.status = status
    if (next.type) query.type = next.type
    if (next.project) query.project = next.project
    if (next.q) query.q = next.q
    if (next.sort !== 'updated') query.sort = next.sort
    void router.replace({ query })
  }

  return { state, apiFilter, update }
}

// Client-side sort. The API only orders by updated_at (or relevance for
// q searches); at personal scale re-sorting the fetched page is enough.
export function sortDocuments(docs: Document[], sort: SortKey): Document[] {
  const out = [...docs]
  switch (sort) {
    case 'title':
      out.sort((a, b) => a.title.localeCompare(b.title))
      break
    case 'created':
      out.sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at))
      break
    default:
      out.sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at))
  }
  return out
}
