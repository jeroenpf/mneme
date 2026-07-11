import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { renderDiagram } from '@/lib/mermaid'
import MDiagram from './MDiagram.vue'

vi.mock('@/lib/mermaid', () => ({ renderDiagram: vi.fn() }))

describe('MDiagram', () => {
  it('renders the svg returned by renderDiagram, with title', async () => {
    vi.mocked(renderDiagram).mockResolvedValue('<svg data-test="mmd"></svg>')
    const w = mount(MDiagram, {
      props: { id: 'ar-d1', title: 'Request flow', content: 'flowchart LR\n A --> B' },
    })
    await flushPromises()
    expect(w.text()).toContain('Request flow')
    expect(w.find('[data-test="mmd"]').exists()).toBe(true)
    expect(vi.mocked(renderDiagram)).toHaveBeenCalledWith(expect.any(String), 'flowchart LR\n A --> B')
  })

  it('falls back to the raw source when rendering fails', async () => {
    vi.mocked(renderDiagram).mockRejectedValue(new Error('parse error'))
    const w = mount(MDiagram, { props: { content: 'not mermaid' } })
    await flushPromises()
    expect(w.find('[data-test="diagram-error"]').text()).toContain('diagram failed to render')
    expect(w.find('pre').text()).toContain('not mermaid')
  })
})
