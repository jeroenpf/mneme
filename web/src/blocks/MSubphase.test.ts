import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MSubphase from './MSubphase.vue'

describe('MSubphase', () => {
  it('renders num badge, session, title, description, tasks, and children', () => {
    const w = mount(MSubphase, {
      props: {
        id: 'sp-1-7',
        num: '1.7',
        title: 'Document viewer',
        session: 7,
        description: 'Renders `body.sections`.',
        tasks: [{ id: 't1', title: 'shell', done: true }],
        children: [{ type: 'text', id: 'p1', content: 'note' }],
      },
    })
    expect(w.attributes('id')).toBe('sp-1-7')
    expect(w.text()).toContain('1.7')
    expect(w.text()).toContain('session 7')
    expect(w.text()).toContain('Document viewer')
    expect(w.html()).toContain('<code>body.sections</code>')
    expect(w.findAll('li')).toHaveLength(1)
    expect(w.text()).toContain('note')
  })

  it('omits session and task list when absent', () => {
    const w = mount(MSubphase, { props: { num: '2', title: 'Swap' } })
    expect(w.text()).not.toContain('session')
    expect(w.find('ul').exists()).toBe(false)
  })

  it('renders a top-level subphase as a numbered masthead (mn-h2)', () => {
    const w = mount(MSubphase, { props: { num: '1', title: 'Backend foundation' } })
    expect(w.find('.sec-head h2').classes()).toContain('mn-h2')
    expect(w.find('.sec-num').text()).toBe('1')
    expect(w.find('.sub-head').exists()).toBe(false)
  })

  it('demotes a nested subphase to a lighter mn-h3 subheading', () => {
    // A subphase inside a parent section (e.g. a phase under "Implementation
    // phases") must read as subordinate, not at the same weight.
    const w = mount(MSubphase, {
      props: { num: '1', title: 'Backend foundation' },
      global: { provide: { 'mn-section-depth': 1 } },
    })
    expect(w.find('h2').exists()).toBe(false)
    expect(w.find('.sub-head h3').classes()).toContain('mn-h3')
    expect(w.find('.sec-head').exists()).toBe(false)
    expect(w.find('.sub-num').text()).toBe('1')
  })

  it('renders a blank-line-separated description as one paragraph per block', () => {
    const w = mount(MSubphase, {
      props: {
        num: '1',
        title: 'P',
        description: '**Files:** a.go.\n\n**Outcome:** works.\n\n**AC:** green.',
      },
    })
    const paras = w.findAll('.mn-prose p')
    expect(paras).toHaveLength(3)
    expect(paras[0]!.html()).toContain('<strong>Files:</strong>')
    expect(paras[2]!.html()).toContain('<strong>AC:</strong>')
  })

  it('sets the session line under the title as the last masthead item', () => {
    // Free-text session notes are long; inline they starve the heading column
    // and wrap the title into a ragged stack, so the masthead drops them to a
    // row of their own.
    const w = mount(MSubphase, {
      props: {
        id: 'sp-9',
        num: '9',
        title: 'Sessions and Activity',
        session: 'Clean session. Branch `feat/admin-sessions-activity` off hyperion/main.',
      },
    })
    const head = w.find('.sec-head')
    expect(head.element.lastElementChild?.classList.contains('session')).toBe(true)
    expect(w.find('.session').text()).toBe(
      'Clean session. Branch `feat/admin-sessions-activity` off hyperion/main.',
    )
  })

  it('sets the session line in sentence-case mono, not a tracked uppercase label', () => {
    // mn-label is uppercase with letter-spacing — right for a two-word tag,
    // unreadable for a sentence.
    const w = mount(MSubphase, { props: { num: '9', title: 'P', session: 'Clean session.' } })
    const session = w.find('.session')
    expect(session.classes()).toContain('mn-mono-sm')
    expect(session.classes()).not.toContain('mn-label')
  })

  it('prefixes a bare number with the word, and leaves free text alone', () => {
    // "session 7" needs the word to mean anything; prefixing a sentence that
    // already names itself reads as "session Clean session."
    const numbered = mount(MSubphase, { props: { num: '1', title: 'P', session: 7 } })
    expect(numbered.find('.session').text()).toBe('session 7')
    const text = mount(MSubphase, { props: { num: '1', title: 'P', session: 'Clean session.' } })
    expect(text.find('.session').text()).toBe('Clean session.')
  })

  it('drops the session line to the same row treatment when nested', () => {
    const w = mount(MSubphase, {
      props: { num: '1', title: 'P', session: 'Clean session.' },
      global: { provide: { 'mn-section-depth': 1 } },
    })
    const head = w.find('.sub-head')
    expect(head.element.lastElementChild?.classList.contains('session')).toBe(true)
  })
})
