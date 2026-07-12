import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import JournalEntryCard from './JournalEntryCard.vue'
import type { JournalEntry } from '@/api/journal'

const entry = (over: Partial<JournalEntry> = {}): JournalEntry => ({
  id: 'e1',
  project: 'apollo',
  session_ref: 'sp-2-4',
  summary: 'Built the journal store',
  accomplished: ['migration', 'store methods'],
  deferred: ['vue timeline'],
  created_at: '2026-07-11T00:00:00Z',
  updated_at: '2026-07-11T00:00:00Z',
  ...over,
})

describe('JournalEntryCard', () => {
  it('renders session_ref, date, summary, and the lists', () => {
    const w = mount(JournalEntryCard, { props: { entry: entry() } })
    expect(w.text()).toContain('Built the journal store')
    expect(w.get('[data-test="session-ref"]').text()).toContain('sp-2-4')
    expect(w.get('[data-test="date"]').text()).toBe('2026-07-11')
    const done = w.findAll('[data-test="accomplished"]')
    expect(done).toHaveLength(2)
    expect(done[0].text()).toContain('migration')
    expect(w.findAll('[data-test="deferred"]')).toHaveLength(1)
  })

  it('omits the lists when empty', () => {
    const w = mount(JournalEntryCard, { props: { entry: entry({ accomplished: [], deferred: [] }) } })
    expect(w.findAll('[data-test="accomplished"]')).toHaveLength(0)
    expect(w.findAll('[data-test="deferred"]')).toHaveLength(0)
  })
})
