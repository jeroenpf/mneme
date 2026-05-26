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

const REGISTRY: Record<string, Component> = {
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

// Dispatcher uses imported Component references (not string names) so
// no global registration is needed. The spec mentioned "register globally"
// because it assumed name-based dispatch; passing refs is cleaner and
// type-safer.
export function typeToComponent(type: string | undefined): Component {
  if (!type) return MText
  return REGISTRY[type] ?? MText
}
