import { inject, provide, ref, type InjectionKey, type Ref } from 'vue'

// The owning document's public id, provided by DocumentView and injected by the
// block renderers so a block/task copy control can build a nested mneme://
// reference without threading the id through every block prop. Undefined when a
// block renders outside a document (e.g. a snippet embedding MCode).
export const DOC_PUBLIC_ID: InjectionKey<Ref<string | undefined>> = Symbol('mn-doc-public-id')

export function provideDocPublicId(publicId: Ref<string | undefined>): void {
  provide(DOC_PUBLIC_ID, publicId)
}

export function useDocPublicId(): Ref<string | undefined> {
  return inject(DOC_PUBLIC_ID, ref<string | undefined>(undefined))
}
