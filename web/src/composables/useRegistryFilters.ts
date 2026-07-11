import { computed, type ComputedRef } from 'vue'
import { useRoute, useRouter, type LocationQueryValue } from 'vue-router'
import type { Document, DocumentFilter, DocumentStatus, DocumentType } from '@/types'

export type SortKey = 'updated' | 'created' | 'title'

export interface RegistryFilterState {
  status?: DocumentStatus
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

const STATUSES: readonly DocumentStatus[] = ['todo', 'in-progress', 'complete', 'blocked', 'archived']
const TYPES: readonly DocumentType[] = ['plan', 'report', 'spec', 'adr', 'brainstorm', 'journal']
const SORTS: readonly SortKey[] = ['updated', 'created', 'title']

function first(v: LocationQueryValue | LocationQueryValue[] | undefined): string | undefined {
  const s = Array.isArray(v) ? v[0] : v
  return typeof s === 'string' && s !== '' ? s : undefined
}

function pick<T extends string>(raw: string | undefined, allowed: readonly T[]): T | undefined {
  return allowed.includes(raw as T) ? (raw as T) : undefined
}

// Filter state lives in the URL query (?status=&type=&project=&q=&sort=)
// so views are linkable and survive reload. sort is client-side only and
// never reaches the API.
export function useRegistryFilters(): UseRegistryFiltersResult {
  const route = useRoute()
  const router = useRouter()

  const state = computed<RegistryFilterState>(() => ({
    status: pick(first(route.query.status), STATUSES),
    type: pick(first(route.query.type), TYPES),
    project: first(route.query.project),
    q: first(route.query.q),
    sort: pick(first(route.query.sort), SORTS) ?? 'updated',
  }))

  const apiFilter = computed<DocumentFilter>(() => {
    const s = state.value
    const f: DocumentFilter = {}
    if (s.status) f.status = s.status
    if (s.type) f.type = s.type
    if (s.project) f.project = s.project
    if (s.q) f.q = s.q
    return f
  })

  function update(patch: Partial<RegistryFilterState>) {
    const next = { ...state.value, ...patch }
    const query: Record<string, string> = {}
    if (next.status) query.status = next.status
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
