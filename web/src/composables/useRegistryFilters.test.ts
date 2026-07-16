import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'
import type { Document } from '@/types'
import { sortDocuments, useRegistryFilters, type UseRegistryFiltersResult } from './useRegistryFilters'

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: defineComponent({ render: () => h('div') }) }],
  })
}

async function withFilters(query = ''): Promise<[UseRegistryFiltersResult, Router]> {
  const router = makeRouter()
  await router.push(`/${query}`)
  await router.isReady()
  let result!: UseRegistryFiltersResult
  mount(
    defineComponent({
      setup() {
        result = useRegistryFilters()
        return () => h('div')
      },
    }),
    { global: { plugins: [router] } },
  )
  return [result, router]
}

describe('useRegistryFilters', () => {
  it('defaults statuses to the working trio when ?status is absent', async () => {
    const [{ state }] = await withFilters('')
    expect(state.value).toEqual({
      statuses: ['todo', 'in-progress', 'blocked'],
      type: undefined,
      project: undefined,
      q: undefined,
      sort: 'updated',
    })
  })

  it('parses a single status and the other typed params', async () => {
    const [{ state }] = await withFilters('?status=complete&type=plan&project=mneme&q=zigbee&sort=title')
    expect(state.value).toEqual({
      statuses: ['complete'],
      type: 'plan',
      project: 'mneme',
      q: 'zigbee',
      sort: 'title',
    })
  })

  it('parses status=none as an empty selection', async () => {
    const [{ state }] = await withFilters('?status=none')
    expect(state.value.statuses).toEqual([])
  })

  it('parses a csv into canonical pill order, dropping duplicates and unknown values', async () => {
    const [{ state }] = await withFilters('?status=blocked,todo,bogus,todo')
    expect(state.value.statuses).toEqual(['todo', 'blocked'])
  })

  it('drops invalid enum values (status=bogus becomes an empty selection)', async () => {
    const [{ state }] = await withFilters('?status=bogus&type=nope&sort=wat')
    expect(state.value).toEqual({
      statuses: [],
      type: undefined,
      project: undefined,
      q: undefined,
      sort: 'updated',
    })
  })

  it('excludes status and sort from the api filter', async () => {
    const [{ apiFilter }] = await withFilters('?status=todo&type=plan&sort=title')
    expect(apiFilter.value).toEqual({ type: 'plan' })
  })

  it('update() omits status when it equals the default trio', async () => {
    const [{ update }, router] = await withFilters('?status=blocked&q=zigbee')
    update({ statuses: ['todo', 'in-progress', 'blocked'] })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ q: 'zigbee' })
  })

  it('update() writes status=none for an empty selection', async () => {
    const [{ update }, router] = await withFilters('')
    update({ statuses: [] })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ status: 'none' })
  })

  it('update() writes a partial selection as a canonical csv', async () => {
    const [{ update }, router] = await withFilters('')
    update({ statuses: ['complete', 'todo'] })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ status: 'todo,complete' })
  })

  it('update() merges a patch into the url, dropping empties and default sort', async () => {
    const [{ update }, router] = await withFilters('?status=todo')
    update({ type: 'plan', q: '' })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ status: 'todo', type: 'plan' })
  })
})

describe('sortDocuments', () => {
  const doc = (id: string, over: Partial<Document>): Document => ({
    id,
    title: id,
    type: 'plan',
    status: 'todo',
    tags: [],
    meta: {},
    body: {},
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...over,
  })

  it('sorts by updated_at desc by default', () => {
    const docs = [
      doc('old', { updated_at: '2026-01-01T00:00:00Z' }),
      doc('new', { updated_at: '2026-07-01T00:00:00Z' }),
    ]
    expect(sortDocuments(docs, 'updated').map((d) => d.id)).toEqual(['new', 'old'])
  })

  it('sorts by created_at desc', () => {
    const docs = [
      doc('old', { created_at: '2026-01-01T00:00:00Z' }),
      doc('new', { created_at: '2026-07-01T00:00:00Z' }),
    ]
    expect(sortDocuments(docs, 'created').map((d) => d.id)).toEqual(['new', 'old'])
  })

  it('sorts by title ascending and does not mutate the input', () => {
    const docs = [doc('b', { title: 'beta' }), doc('a', { title: 'alpha' })]
    expect(sortDocuments(docs, 'title').map((d) => d.id)).toEqual(['a', 'b'])
    expect(docs.map((d) => d.id)).toEqual(['b', 'a'])
  })
})
