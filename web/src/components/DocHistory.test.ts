import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { listRevisions, restoreRevision, type DocRevision } from '@/api/documents'
import { useToast } from '@/composables/useToast'
import DocHistory from './DocHistory.vue'

vi.mock('@/api/documents', () => ({ listRevisions: vi.fn(), restoreRevision: vi.fn() }))

const revs: DocRevision[] = [
  { revision: 3, op: 'rest:update', actor: 'rest', target_ids: ['s1'], title: 'Doc', status: 'in-progress', created_at: new Date(Date.now() - 60_000).toISOString() },
  { revision: 2, op: 'tick_task', actor: 'mcp', target_ids: ['t1'], title: 'Doc', status: 'in-progress', created_at: new Date(Date.now() - 3_600_000).toISOString() },
  { revision: 1, op: 'rest:create', actor: 'rest', target_ids: [], title: 'Doc', status: 'todo', created_at: new Date(Date.now() - 86_400_000).toISOString() },
]

function mountHistory(currentRevision = 3) {
  return mount(DocHistory, {
    props: { docId: 'my-doc', currentRevision },
    attachTo: document.body,
  })
}

beforeEach(() => {
  vi.mocked(listRevisions).mockReset().mockResolvedValue(revs)
  vi.mocked(restoreRevision).mockReset().mockResolvedValue({
    restored_from: 1,
    new_revision: 4,
    doc: {} as never,
  })
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('DocHistory', () => {
  it('is collapsed until toggled, then loads revisions', async () => {
    const w = mountHistory()
    await flushPromises()
    expect(w.find('[data-test="rev-row"]').exists()).toBe(false)
    expect(listRevisions).not.toHaveBeenCalled()

    await w.get('[data-test="history-toggle"]').trigger('click')
    await flushPromises()
    expect(listRevisions).toHaveBeenCalledWith('my-doc')
    expect(w.findAll('[data-test="rev-row"]')).toHaveLength(3)
  })

  it('shows each revision with its op and actor', async () => {
    const w = mountHistory()
    await w.get('[data-test="history-toggle"]').trigger('click')
    await flushPromises()
    const rows = w.findAll('[data-test="rev-row"]')
    expect(rows[0].text()).toContain('rest:update')
    expect(rows[1].text()).toContain('tick_task')
    expect(rows[1].text()).toContain('mcp')
  })

  it('offers restore on past revisions but not the current one', async () => {
    const w = mountHistory(3)
    await w.get('[data-test="history-toggle"]').trigger('click')
    await flushPromises()
    const rows = w.findAll('[data-test="rev-row"]')
    // Row 0 is revision 3 = current → no restore; rows 1 and 2 restorable.
    expect(rows[0].find('[data-test="restore"]').exists()).toBe(false)
    expect(rows[1].find('[data-test="restore"]').exists()).toBe(true)
    expect(rows[2].find('[data-test="restore"]').exists()).toBe(true)
  })

  it('restores a revision, toasts, and emits restored', async () => {
    const w = mountHistory(3)
    await w.get('[data-test="history-toggle"]').trigger('click')
    await flushPromises()
    await w.findAll('[data-test="rev-row"]')[2].get('[data-test="restore"]').trigger('click')
    await flushPromises()
    expect(restoreRevision).toHaveBeenCalledWith('my-doc', 1)
    expect(useToast().toasts.some((t) => /restored/i.test(t.message))).toBe(true)
    expect(w.emitted('restored')).toBeTruthy()
  })
})
