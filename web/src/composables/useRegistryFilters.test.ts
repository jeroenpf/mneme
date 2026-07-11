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
  it('parses valid query params into typed state', async () => {
    const [{ state }] = await withFilters('?status=complete&type=plan&project=mneme&q=zigbee&sort=title')
    expect(state.value).toEqual({
      status: 'complete',
      type: 'plan',
      project: 'mneme',
      q: 'zigbee',
      sort: 'title',
    })
  })

  it('drops invalid enum values and defaults sort to updated', async () => {
    const [{ state }] = await withFilters('?status=bogus&type=nope&sort=wat')
    expect(state.value).toEqual({
      status: undefined,
      type: undefined,
      project: undefined,
      q: undefined,
      sort: 'updated',
    })
  })

  it('excludes sort from the api filter', async () => {
    const [{ apiFilter }] = await withFilters('?status=todo&sort=title')
    expect(apiFilter.value).toEqual({ status: 'todo' })
  })

  it('update() merges a patch into the url, dropping empties and default sort', async () => {
    const [{ update }, router] = await withFilters('?status=todo')
    update({ type: 'plan', q: '' })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ status: 'todo', type: 'plan' })
  })

  it('update() can clear a previously set param', async () => {
    const [{ update }, router] = await withFilters('?status=todo&q=zigbee')
    update({ status: undefined })
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.query).toEqual({ q: 'zigbee' })
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
