import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reindexFailed, searchStatus, type SearchStatus } from '@/api/search'
import { useToast } from '@/composables/useToast'
import EmbeddingsView from './EmbeddingsView.vue'

vi.mock('@/api/search', () => ({ searchStatus: vi.fn(), reindexFailed: vi.fn() }))

const makeStatus = (over: Partial<SearchStatus> = {}): SearchStatus => ({
  enabled: true,
  provider: { name: 'voyage', model: 'voyage-4-large', enabled: true },
  queue_depth: 2,
  last_reconcile: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  items: [
    { type: 'documents', total: 10, embedded: 8, reconciled: 8, missing: 2, stale: 0, orphaned: 0, failed: 1 },
    { type: 'decisions', total: 4, embedded: 4, reconciled: 4, missing: 0, stale: 0, orphaned: 0, failed: 0 },
  ],
  ...over,
})

async function mountView() {
  const w = mount(EmbeddingsView, { attachTo: document.body })
  await flushPromises()
  return w
}

beforeEach(() => {
  vi.mocked(searchStatus).mockReset().mockResolvedValue(makeStatus())
  vi.mocked(reindexFailed).mockReset().mockResolvedValue({ retried: 1 })
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('EmbeddingsView', () => {
  it('shows the provider name and model when enabled', async () => {
    const w = await mountView()
    const provider = w.get('[data-test="provider"]')
    expect(provider.text()).toContain('voyage')
    expect(provider.text()).toContain('voyage-4-large')
  })

  it('shows queue depth and the last reconciliation time', async () => {
    const w = await mountView()
    expect(w.get('[data-test="queue-depth"]').text()).toContain('2')
    expect(w.get('[data-test="last-reconcile"]').text().toLowerCase()).toMatch(/ago|now/)
  })

  it('renders a coverage row per source type with buckets', async () => {
    const w = await mountView()
    const rows = w.findAll('[data-test="type-row"]')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('documents')
    expect(rows[0].text()).toContain('8') // embedded
    expect(rows[0].text()).toContain('10') // total
  })

  it('retries failed embeddings and reports the count', async () => {
    const w = await mountView()
    const btn = w.get('[data-test="retry-failed"]')
    await btn.trigger('click')
    await flushPromises()
    expect(reindexFailed).toHaveBeenCalledOnce()
    expect(useToast().toasts.some((t) => /1/.test(t.message))).toBe(true)
    // Re-fetches status after a retry (mount + refresh).
    expect(searchStatus).toHaveBeenCalledTimes(2)
  })

  it('hides the retry control when nothing has failed', async () => {
    vi.mocked(searchStatus).mockResolvedValue(
      makeStatus({
        items: [
          { type: 'documents', total: 10, embedded: 10, reconciled: 10, missing: 0, stale: 0, orphaned: 0, failed: 0 },
        ],
      }),
    )
    const w = await mountView()
    expect(w.find('[data-test="retry-failed"]').exists()).toBe(false)
  })

  it('shows a lexical-only notice when embeddings are disabled', async () => {
    vi.mocked(searchStatus).mockResolvedValue(
      makeStatus({ enabled: false, provider: { name: '', model: '', enabled: false } }),
    )
    const w = await mountView()
    expect(w.get('[data-test="disabled"]').text().toLowerCase()).toContain('lexical')
    expect(w.find('[data-test="retry-failed"]').exists()).toBe(false)
  })
})
