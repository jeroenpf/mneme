export interface NavItem {
  id: string
  title: string
  kind: 'section' | 'subphase'
  num?: string
}

// Sidebar nav lists top-level sections/subphases only — nesting stays a
// content concern. Blocks need both id and title to be navigable.
export function sectionNavItems(body: Record<string, unknown> | undefined): NavItem[] {
  const sections = body?.sections
  if (!Array.isArray(sections)) return []
  const out: NavItem[] = []
  for (const raw of sections) {
    if (typeof raw !== 'object' || raw === null) continue
    const b = raw as { type?: unknown; id?: unknown; title?: unknown; num?: unknown }
    if (b.type !== 'section' && b.type !== 'subphase') continue
    if (typeof b.id !== 'string' || typeof b.title !== 'string') continue
    out.push({
      id: b.id,
      title: b.title,
      kind: b.type,
      num: typeof b.num === 'string' ? b.num : undefined,
    })
  }
  return out
}
