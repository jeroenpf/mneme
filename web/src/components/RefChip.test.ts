import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RefChip from './RefChip.vue'
import { useToast } from '@/composables/useToast'

function stubClipboard() {
  const writeText = vi.fn().mockResolvedValue(undefined)
  vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
  return writeText
}

describe('RefChip', () => {
  it('shows the public id and copies it, raising a toast', async () => {
    const writeText = stubClipboard()
    const { toasts } = useToast()
    const before = toasts.length
    const w = mount(RefChip, { props: { publicId: 'doc_000000000000', kind: 'document' } })

    expect(w.get('[data-test="copy-id"]').text()).toContain('doc_000000000000')
    await w.get('[data-test="copy-id"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('doc_000000000000')
    expect(toasts.length).toBe(before + 1)
    expect(toasts[toasts.length - 1].message).toContain('doc_000000000000')
  })

  it('copies the canonical reference for a top-level entity', async () => {
    const writeText = stubClipboard()
    const w = mount(RefChip, { props: { publicId: 'dec_000000000000', kind: 'decision' } })
    await w.get('[data-test="copy-ref"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('mneme://decision/dec_000000000000')
  })

  it('compact mode omits the id text and only copies the reference', async () => {
    const writeText = stubClipboard()
    const w = mount(RefChip, {
      props: { publicId: 'blk_111111111111', kind: 'block', docId: 'doc_000000000000', compact: true },
    })
    expect(w.find('[data-test="copy-id"]').exists()).toBe(false)
    await w.get('[data-test="copy-ref"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('mneme://document/doc_000000000000/block/blk_111111111111')
  })
})
