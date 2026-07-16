export interface NavItem {
  id: string
  title: string
  kind: 'section' | 'subphase'
  /** Zero-padded sequential position in the TOC (01, 02, …). Always set. */
  num: string
}

// Sidebar nav lists top-level sections/subphases only — nesting stays a
// content concern. Blocks need both id and title to be navigable. Each entry
// is numbered by its position so the TOC reads 01…NN in document order; the
// body keeps its own markers (a section's sequential number, a subphase's
// phase badge), which DocumentView derives from this same list.
export function sectionNavItems(body: Record<string, unknown> | undefined): NavItem[] {
  const sections = body?.sections
  if (!Array.isArray(sections)) return []
  const out: NavItem[] = []
  for (const raw of sections) {
    if (typeof raw !== 'object' || raw === null) continue
    const b = raw as { type?: unknown; id?: unknown; title?: unknown }
    if (b.type !== 'section' && b.type !== 'subphase') continue
    if (typeof b.id !== 'string' || typeof b.title !== 'string') continue
    out.push({
      id: b.id,
      title: b.title,
      kind: b.type,
      num: String(out.length + 1).padStart(2, '0'),
    })
  }
  return out
}
