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
  it('renders one control per theme', () => {
    const w = mount(ThemePicker)
    expect(w.findAll('button[data-test^="theme-"]')).toHaveLength(3)
  })

  it('applies and persists the theme when a control is clicked', async () => {
    const w = mount(ThemePicker)
    await w.find('[data-test="theme-ink"]').trigger('click')
    expect(document.documentElement.dataset.theme).toBe('ink')
    expect(localStorage.getItem('mneme.theme')).toBe('ink')
  })

  it('marks the active theme with aria-pressed', async () => {
    const w = mount(ThemePicker)
    await w.find('[data-test="theme-slate"]').trigger('click')
    expect(w.find('[data-test="theme-slate"]').attributes('aria-pressed')).toBe('true')
    expect(w.find('[data-test="theme-paper"]').attributes('aria-pressed')).toBe('false')
  })
})
