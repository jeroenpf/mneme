import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ThemePicker from './ThemePicker.vue'
import { useTheme } from '@/composables/useTheme'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
  // Reset the module-level singleton to the default between cases.
  useTheme().setTheme('paper')
  localStorage.clear()
})

describe('ThemePicker', () => {
  it('renders one swatch per theme', () => {
    const w = mount(ThemePicker)
    expect(w.findAll('button[data-test^="theme-"]')).toHaveLength(4)
  })

  it('applies and persists the theme when a swatch is clicked', async () => {
    const w = mount(ThemePicker)
    await w.find('[data-test="theme-graphite"]').trigger('click')
    expect(document.documentElement.dataset.theme).toBe('graphite')
    expect(localStorage.getItem('mneme.theme')).toBe('graphite')
  })

  it('marks the active theme with aria-pressed', async () => {
    const w = mount(ThemePicker)
    await w.find('[data-test="theme-slate"]').trigger('click')
    expect(w.find('[data-test="theme-slate"]').attributes('aria-pressed')).toBe('true')
    expect(w.find('[data-test="theme-paper"]').attributes('aria-pressed')).toBe('false')
  })

  it('names each swatch for screen readers and tooltips', () => {
    const w = mount(ThemePicker)
    const ink = w.find('[data-test="theme-ink"]')
    expect(ink.attributes('aria-label')).toBe('Ink')
    expect(ink.attributes('title')).toBe('Ink')
  })

  it('shows the active theme name', async () => {
    const w = mount(ThemePicker)
    await w.find('[data-test="theme-graphite"]').trigger('click')
    expect(w.find('[data-test="theme-name"]').text()).toBe('Graphite')
  })
})
