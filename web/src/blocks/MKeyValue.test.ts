import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MKeyValue from './MKeyValue.vue'

describe('MKeyValue', () => {
  it('renders a dt/dd per entry with inline md values', () => {
    const w = mount(MKeyValue, {
      props: { title: 'At a glance', data: { Stack: 'Go + Vue', Domain: '`mneme.local`' } },
    })
    expect(w.text()).toContain('At a glance')
    expect(w.findAll('dt')).toHaveLength(2)
    expect(w.findAll('dd')[1].html()).toContain('<code>mneme.local</code>')
  })

  it('renders nothing without data', () => {
    expect(mount(MKeyValue, { props: {} }).find('dl').exists()).toBe(false)
  })
})
