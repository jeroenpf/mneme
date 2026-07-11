import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ProjectStats } from '@/types'
import type { RegistryFilterState } from '@/composables/useRegistryFilters'
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

function mountToolbar(state: Partial<RegistryFilterState> = {}) {
  return mount(FilterToolbar, {
    props: { state: { sort: 'updated', ...state }, projects },
  })
}

function pill(w: ReturnType<typeof mountToolbar>, label: string) {
  const b = w.findAll('button.pill').find((p) => p.text() === label)
  if (!b) throw new Error(`no pill ${label}`)
  return b
}

describe('FilterToolbar', () => {
  it('marks the matching status pill active, or "all" when unfiltered', () => {
    expect(pill(mountToolbar(), 'all').classes()).toContain('active')
    const w = mountToolbar({ status: 'complete' })
    expect(pill(w, 'complete').classes()).toContain('active')
    expect(pill(w, 'all').classes()).not.toContain('active')
  })

  it('emits a status patch on pill click, and clears on active-pill click', async () => {
    const w = mountToolbar({ status: 'complete' })
    await pill(w, 'todo').trigger('click')
    expect(w.emitted('change')?.[0]).toEqual([{ status: 'todo' }])
    await pill(w, 'complete').trigger('click')
    expect(w.emitted('change')?.[1]).toEqual([{ status: undefined }])
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
})
