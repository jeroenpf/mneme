import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ProjectStats } from '@/types'
import type { RegistryFilterState } from '@/composables/useRegistryFilters'
import type { ViewMode } from '@/composables/useViewMode'
import FilterToolbar from './FilterToolbar.vue'

const projects: ProjectStats[] = [
  {
    id: '1',
    name: 'Mneme',
    slug: 'mneme',
    created_at: '',
    counts: { todo: 0, 'in-progress': 0, complete: 0, blocked: 0, archived: 0, total: 0 },
  },
]

function mountToolbar(state: Partial<RegistryFilterState> = {}, view: ViewMode = 'cards') {
  return mount(FilterToolbar, {
    props: { state: { sort: 'updated', statuses: [], ...state }, projects, view },
  })
}

function pill(w: ReturnType<typeof mountToolbar>, label: string) {
  const b = w.findAll('button.pill').find((p) => p.text() === label)
  if (!b) throw new Error(`no pill ${label}`)
  return b
}

describe('FilterToolbar', () => {
  it('marks each selected status pill active, and "all" only when all four are selected', () => {
    const trio = mountToolbar({ statuses: ['todo', 'in-progress', 'blocked'] })
    expect(pill(trio, 'todo').classes()).toContain('active')
    expect(pill(trio, 'complete').classes()).not.toContain('active')
    expect(pill(trio, 'all').classes()).not.toContain('active')

    const all = mountToolbar({ statuses: ['todo', 'in-progress', 'complete', 'blocked'] })
    expect(pill(all, 'all').classes()).toContain('active')
  })

  it('toggles a status into/out of the selection in canonical order', async () => {
    const w = mountToolbar({ statuses: ['todo', 'in-progress', 'blocked'] })
    await pill(w, 'complete').trigger('click')
    expect(w.emitted('change')?.[0]).toEqual([
      { statuses: ['todo', 'in-progress', 'complete', 'blocked'] },
    ])
    await pill(w, 'in-progress').trigger('click')
    expect(w.emitted('change')?.[1]).toEqual([{ statuses: ['todo', 'blocked'] }])
  })

  it('selects all four statuses when "all" is clicked', async () => {
    const w = mountToolbar({ statuses: ['todo'] })
    await pill(w, 'all').trigger('click')
    expect(w.emitted('change')?.[0]).toEqual([
      { statuses: ['todo', 'in-progress', 'complete', 'blocked'] },
    ])
  })

  it('lists projects in the dropdown and emits a project patch', async () => {
    const w = mountToolbar()
    const select = w.find('select[aria-label="Filter by project"]')
    expect(select.findAll('option').map((o) => o.text())).toEqual(['all projects', 'mneme'])
    await select.setValue('mneme')
    expect(w.emitted('change')?.[0]).toEqual([{ project: 'mneme' }])
  })

  it('emits type and sort patches from the selects', async () => {
    const w = mountToolbar()
    await w.find('select[aria-label="Filter by type"]').setValue('adr')
    expect(w.emitted('change')?.[0]).toEqual([{ type: 'adr' }])
    await w.find('select[aria-label="Sort by"]').setValue('title')
    expect(w.emitted('change')?.[1]).toEqual([{ sort: 'title' }])
  })

  it('marks the active view button from the view prop', () => {
    const cards = mountToolbar({}, 'cards')
    expect(cards.find('[data-test="view-cards"]').classes()).toContain('active')
    expect(cards.find('[data-test="view-list"]').classes()).not.toContain('active')

    const list = mountToolbar({}, 'list')
    expect(list.find('[data-test="view-list"]').classes()).toContain('active')
    expect(list.find('[data-test="view-cards"]').classes()).not.toContain('active')
  })

  it('emits viewChange with the clicked mode', async () => {
    const w = mountToolbar({}, 'cards')
    await w.find('[data-test="view-list"]').trigger('click')
    expect(w.emitted('viewChange')?.[0]).toEqual(['list'])
    await w.find('[data-test="view-cards"]').trigger('click')
    expect(w.emitted('viewChange')?.[1]).toEqual(['cards'])
  })
})
