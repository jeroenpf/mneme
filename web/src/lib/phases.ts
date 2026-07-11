export interface DocPhase {
  title: string
  status: 'done' | 'wip' | 'todo'
}

// meta is untyped JSONB from the API — extract defensively. Entries
// without a title are dropped; unknown statuses degrade to todo.
export function phasesFromMeta(meta: Record<string, unknown> | undefined): DocPhase[] {
  const raw = meta?.phases
  if (!Array.isArray(raw)) return []
  const out: DocPhase[] = []
  for (const entry of raw) {
    if (typeof entry !== 'object' || entry === null) continue
    const { title, status } = entry as { title?: unknown; status?: unknown }
    if (typeof title !== 'string' || title === '') continue
    out.push({
      title,
      status: status === 'done' || status === 'wip' ? status : 'todo',
    })
  }
  return out
}
