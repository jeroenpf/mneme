import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import BlockRefCopy from './BlockRefCopy.vue'
import { DOC_PUBLIC_ID } from '@/composables/useDocRef'

function stubClipboard() {
  const writeText = vi.fn().mockResolvedValue(undefined)
  vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
  return writeText
}

function mountWith(docId: string | undefined, props: { blockId?: string; kind: 'block' | 'task' }) {
  return mount(BlockRefCopy, {
    props,
    global: { provide: { [DOC_PUBLIC_ID as symbol]: ref(docId) } },
  })
}

describe('BlockRefCopy', () => {
  it('copies the block reference nested under the injected document', async () => {
    const writeText = stubClipboard()
    const w = mountWith('doc_000000000000', { blockId: 'blk_111111111111', kind: 'block' })
    await w.get('[data-test="copy-ref"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('mneme://document/doc_000000000000/block/blk_111111111111')
  })

  it('copies a task reference (semantic ids allowed)', async () => {
    const writeText = stubClipboard()
    const w = mountWith('doc_000000000000', { blockId: 's6-t1', kind: 'task' })
    await w.get('[data-test="copy-ref"]').trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith('mneme://document/doc_000000000000/task/s6-t1')
  })

  it('renders nothing without a document public id in scope', () => {
    const w = mountWith(undefined, { blockId: 'blk_1', kind: 'block' })
    expect(w.find('[data-test="copy-ref"]').exists()).toBe(false)
  })

  it('renders nothing when the block has no id', () => {
    const w = mountWith('doc_000000000000', { blockId: undefined, kind: 'block' })
    expect(w.find('[data-test="copy-ref"]').exists()).toBe(false)
  })
})
