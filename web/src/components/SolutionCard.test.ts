import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SolutionCard from './SolutionCard.vue'
import type { Solution } from '@/api/solutions'

const sol = (over: Partial<Solution> = {}): Solution => ({
  id: 's1',
  project: 'apollo',
  error_description: 'container startup timeout',
  solution: 'raise the healthcheck start_period',
  tags: ['docker', 'compose'],
  source_url: 'https://example.test/fix',
  created_at: '2026-07-11T00:00:00Z',
  updated_at: '2026-07-11T00:00:00Z',
  ...over,
})

describe('SolutionCard', () => {
  it('renders error, solution, tags, source link, and date', () => {
    const w = mount(SolutionCard, { props: { solution: sol() } })
    expect(w.get('[data-test="error"]').text()).toContain('container startup timeout')
    expect(w.get('[data-test="solution"]').text()).toContain('raise the healthcheck start_period')
    expect(w.findAll('[data-test="tag"]')).toHaveLength(2)
    expect(w.get('[data-test="date"]').text()).toBe('2026-07-11')
    const link = w.get('[data-test="source-url"]')
    expect(link.attributes('href')).toBe('https://example.test/fix')
  })

  it('omits the source link and tags when empty', () => {
    const w = mount(SolutionCard, { props: { solution: sol({ tags: [], source_url: '' }) } })
    expect(w.findAll('[data-test="tag"]')).toHaveLength(0)
    expect(w.findAll('[data-test="source-url"]')).toHaveLength(0)
  })
})
