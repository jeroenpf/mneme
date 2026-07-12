import { describe, expect, it } from 'vitest'
import type { MemoryEntry } from '@/api/memory'
import { groupMemory } from './useMemory'

const entry = (over: Partial<MemoryEntry> & Pick<MemoryEntry, 'scope' | 'key' | 'value'>): MemoryEntry => ({
  id: `${over.scope}:${over.project ?? ''}:${over.area ?? ''}:${over.key}`,
  updated_at: '2026-07-01T00:00:00Z',
  ...over,
})

describe('groupMemory', () => {
  const entries: MemoryEntry[] = [
    entry({ scope: 'global', key: 'shell', value: 'zsh' }),
    entry({ scope: 'global', key: 'editor', value: 'vscode' }),
    entry({ scope: 'project', project: 'hermes', key: 'lang', value: 'rust' }),
    entry({ scope: 'project', project: 'apollo', key: 'stack', value: 'go' }),
    entry({ scope: 'area', project: 'apollo', area: 'billing', key: 'currency', value: 'usd' }),
  ]

  it('splits global entries out, sorted by key', () => {
    const g = groupMemory(entries)
    expect(g.global.map((e) => e.key)).toEqual(['editor', 'shell'])
  })

  it('groups project + area scopes under their project, sorted by project name', () => {
    const g = groupMemory(entries)
    expect(g.projects.map((p) => p.project)).toEqual(['apollo', 'hermes'])

    const apollo = g.projects.find((p) => p.project === 'apollo')!
    expect(apollo.entries.map((e) => e.key)).toEqual(['stack'])
    expect(apollo.areas.map((a) => a.area)).toEqual(['billing'])
    expect(apollo.areas[0].entries.map((e) => e.key)).toEqual(['currency'])

    const hermes = g.projects.find((p) => p.project === 'hermes')!
    expect(hermes.entries.map((e) => e.key)).toEqual(['lang'])
    expect(hermes.areas).toEqual([])
  })

  it('returns empty groups for no entries', () => {
    const g = groupMemory([])
    expect(g.global).toEqual([])
    expect(g.projects).toEqual([])
  })
})
