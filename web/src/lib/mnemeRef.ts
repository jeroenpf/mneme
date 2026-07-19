// mnemeRef.ts — client-side mirror of the Go reference grammar
// (internal/ids/refs.go). Formats mneme:// references so the UI can build
// copyable references, and parses pasted ones so navigation can open them.
// Permissive on nested child ids so semantic ids (pre blk_/task_ migration)
// still parse; the backend resolver does the strict validation.

export type Kind =
  | 'project'
  | 'document'
  | 'block'
  | 'task'
  | 'decision'
  | 'journal'
  | 'snippet'
  | 'solution'

export interface MnemeRef {
  kind: Kind
  id: string
  docId?: string
}

const SCHEME = 'mneme://'

// prefix → kind, mirroring internal/ids/ids.go kindPrefix.
const PREFIX_KIND: Record<string, Kind> = {
  prj: 'project',
  doc: 'document',
  blk: 'block',
  task: 'task',
  dec: 'decision',
  jrnl: 'journal',
  snip: 'snippet',
  sol: 'solution',
}

// kinds addressable as a top-level URI segment (blocks/tasks nest under a doc).
const TOP_LEVEL: Record<string, Kind> = {
  project: 'project',
  document: 'document',
  decision: 'decision',
  journal: 'journal',
  snippet: 'snippet',
  solution: 'solution',
}

// publicIdKind maps a bare public id to its kind by prefix, or null.
export function publicIdKind(id: string): Kind | null {
  const us = id.indexOf('_')
  if (us <= 0) return null
  return PREFIX_KIND[id.slice(0, us)] ?? null
}

// formatRef builds the canonical mneme:// reference. Blocks and tasks nest
// under their owning document, so pass docId for those.
export function formatRef(kind: Kind, id: string, docId?: string): string {
  if (kind === 'block' || kind === 'task') {
    return `${SCHEME}document/${docId}/${kind}/${id}`
  }
  return `${SCHEME}${kind}/${id}`
}

// parseRef parses a canonical mneme:// reference or a bare top-level public id.
// Returns null for anything unrecognisable so callers can branch simply.
export function parseRef(input: string): MnemeRef | null {
  const s = input.trim()
  if (s === '') return null

  if (!s.startsWith(SCHEME)) {
    // Bare public id — its prefix names the kind. Block/task ids need a doc.
    const kind = publicIdKind(s)
    if (!kind || kind === 'block' || kind === 'task') return null
    return { kind, id: s }
  }

  const segs = s.slice(SCHEME.length).split('/')
  if (segs.length === 2) {
    const kind = TOP_LEVEL[segs[0]]
    if (!kind || segs[1] === '') return null
    return { kind, id: segs[1] }
  }
  if (segs.length === 4) {
    const [owner, docId, relation, childId] = segs
    if (owner !== 'document' || docId === '' || childId === '') return null
    if (relation !== 'block' && relation !== 'task') return null
    return { kind: relation, id: childId, docId }
  }
  return null
}
