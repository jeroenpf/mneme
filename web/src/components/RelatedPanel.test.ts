import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'
import RelatedPanel from './RelatedPanel.vue'
import { getRelated, type RelatedBundle } from '@/api/relations'

vi.mock('@/api/relations', () => ({ getRelated: vi.fn() }))

const Blank = defineComponent({ render: () => h('div') })

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Blank },
      { path: '/doc/:id', component: Blank },
      { path: '/decisions', component: Blank },
    ],
  })
}

async function mountPanel(bundle: RelatedBundle) {
  vi.mocked(getRelated).mockResolvedValue(bundle)
  const router = makeRouter()
  await router.push('/')
  const w = mount(RelatedPanel, {
    props: { docId: 'plan-a', revision: 1 },
    global: { plugins: [router] },
  })
  await flushPromises()
  return w
}

const empty: RelatedBundle = { links: [], mentions: [], mentioned_by: [] }

beforeEach(() => {
  vi.mocked(getRelated).mockReset()
})

describe('RelatedPanel', () => {
  it('renders nothing when there are no relations', async () => {
    const w = await mountPanel(empty)
    expect(w.find('[data-test="related-panel"]').exists()).toBe(false)
  })

  it('renders nothing when the endpoint fails', async () => {
    vi.mocked(getRelated).mockRejectedValue(new Error('boom'))
    const router = makeRouter()
    const w = mount(RelatedPanel, { props: { docId: 'plan-a' }, global: { plugins: [router] } })
    await flushPromises()
    expect(w.find('[data-test="related-panel"]').exists()).toBe(false)
  })

  it('labels typed links by direction and routes entries', async () => {
    const w = await mountPanel({
      links: [
        { id: 'doc_spec1', kind: 'document', title: 'Spec X', rel_type: 'implements', direction: 'out', doc_status: 'complete' },
        { id: 'dec_call1', kind: 'decision', title: 'Big call', rel_type: 'depends-on', direction: 'in' },
      ],
      mentions: [],
      mentioned_by: [],
    })
    const links = w.get('[data-test="related-links"]')
    expect(links.text()).toContain('implements')
    expect(links.text()).toContain('Spec X')
    expect(links.text()).toContain('depended on by')
    expect(links.text()).toContain('Big call')
    const anchors = links.findAllComponents({ name: 'RouterLink' })
    expect(anchors[0].props('to')).toEqual({ path: '/doc/doc_spec1' })
    expect(anchors[1].props('to')).toEqual({ path: '/decisions', query: { flash: 'dec_call1' } })
  })

  it('shows backlinks under Mentioned by and dims dangling mentions', async () => {
    const w = await mountPanel({
      links: [],
      mentions: [{ id: 'future-doc', kind: 'document', title: 'future-doc', rel_type: 'mentions', direction: 'out', dangling: true }],
      mentioned_by: [{ id: 'doc_plan9', kind: 'document', title: 'Plan 9', rel_type: 'mentions', direction: 'in' }],
    })
    const back = w.get('[data-test="related-backlinks"]')
    expect(back.text()).toContain('Plan 9')
    const mentions = w.get('[data-test="related-mentions"]')
    expect(mentions.text()).toContain('future-doc')
    expect(mentions.find('.dangling').exists()).toBe(true)
    expect(mentions.findAllComponents({ name: 'RouterLink' })).toHaveLength(0)
  })

  it('refetches when the document revision changes', async () => {
    const w = await mountPanel(empty)
    expect(vi.mocked(getRelated)).toHaveBeenCalledTimes(1)
    await w.setProps({ revision: 2 })
    await flushPromises()
    expect(vi.mocked(getRelated)).toHaveBeenCalledTimes(2)
  })
})
