import type { Component } from 'vue'
import MCallout from './MCallout.vue'
import MCode from './MCode.vue'
import MDiagram from './MDiagram.vue'
import MKeyValue from './MKeyValue.vue'
import MSection from './MSection.vue'
import MSubphase from './MSubphase.vue'
import MTable from './MTable.vue'
import MTaskList from './MTaskList.vue'
import MText from './MText.vue'

// Built lazily on first dispatch: the import cycle MSection →
// BlockRenderer → index → MSection means an eager literal would capture
// `undefined` for whichever component started the cycle, and nested
// sections would silently render nothing.
let registry: Record<string, Component> | null = null

function getRegistry(): Record<string, Component> {
  registry ??= {
    section: MSection,
    subphase: MSubphase,
    'task-list': MTaskList,
    callout: MCallout,
    code: MCode,
    table: MTable,
    diagram: MDiagram,
    'key-value': MKeyValue,
    text: MText,
  }
  return registry
}

// Dispatcher uses imported Component references (not string names) so
// no global registration is needed. The spec mentioned "register globally"
// because it assumed name-based dispatch; passing refs is cleaner and
// type-safer.
export function typeToComponent(type: string | undefined): Component {
  if (!type) return MText
  return getRegistry()[type] ?? MText
}
