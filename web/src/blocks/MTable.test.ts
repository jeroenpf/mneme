import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MTable from './MTable.vue'

describe('MTable', () => {
  it('renders cols as headers and rows as inline-md cells', () => {
    const w = mount(MTable, {
      props: {
        title: 'Endpoints',
        cols: ['Method', 'Path'],
        rows: [['GET', '`/api/v1/documents`'], ['GET', '**bold** path']],
      },
    })
    expect(w.text()).toContain('Endpoints')
    expect(w.findAll('th').map((t) => t.text())).toEqual(['Method', 'Path'])
    expect(w.findAll('tbody tr')).toHaveLength(2)
    expect(w.find('tbody').html()).toContain('<code>/api/v1/documents</code>')
    expect(w.find('tbody').html()).toContain('<strong>bold</strong>')
  })
})
