import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { applyTheme, renderDiagram } from '@/lib/mermaid'
import { useTheme } from '@/composables/useTheme'
import MDiagram from './MDiagram.vue'

vi.mock('@/lib/mermaid', () => ({ renderDiagram: vi.fn(), applyTheme: vi.fn() }))

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
    // Unmount: current is a module-level singleton, so a lingering component
    // would re-run its effect on any later theme change and skew call counts.
    w.unmount()
  })

  it('falls back to the raw source when rendering fails', async () => {
    vi.mocked(renderDiagram).mockRejectedValue(new Error('parse error'))
    const w = mount(MDiagram, { props: { content: 'not mermaid' } })
    await flushPromises()
    expect(w.find('[data-test="diagram-error"]').text()).toContain('diagram failed to render')
    expect(w.find('pre').text()).toContain('not mermaid')
    w.unmount()
  })

  it('re-applies the theme then re-renders when the theme changes', async () => {
    vi.mocked(renderDiagram).mockClear()
    vi.mocked(applyTheme).mockClear()
    vi.mocked(renderDiagram).mockResolvedValue('<svg data-test="mmd"></svg>')

    const w = mount(MDiagram, { props: { id: 'ph-d1', content: 'flowchart LR\n A --> B' } })
    await flushPromises()
    expect(vi.mocked(applyTheme)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(renderDiagram)).toHaveBeenCalledTimes(1)

    const { current, setTheme } = useTheme()
    setTheme(current.value === 'ink' ? 'paper' : 'ink')
    await flushPromises()

    // The theme switch re-runs the effect: re-apply mermaid's theme, then re-render.
    expect(vi.mocked(applyTheme)).toHaveBeenCalledTimes(2)
    expect(vi.mocked(renderDiagram)).toHaveBeenCalledTimes(2)

    // applyTheme is awaited before the render it precedes (fresh init → fresh colors).
    const applyOrder = vi.mocked(applyTheme).mock.invocationCallOrder
    const renderOrder = vi.mocked(renderDiagram).mock.invocationCallOrder
    expect(applyOrder[1]).toBeLessThan(renderOrder[1])

    w.unmount()
  })
})
