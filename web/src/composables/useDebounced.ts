import { ref, watch, type Ref } from 'vue'

// Returns a ref that trails `source` by `delay` ms. Rapid writes
// collapse into one trailing update.
export function useDebounced<T>(source: Ref<T>, delay = 300): Readonly<Ref<T>> {
  const out = ref(source.value) as Ref<T>
  let timer: ReturnType<typeof setTimeout> | undefined
  watch(source, (value) => {
    clearTimeout(timer)
    timer = setTimeout(() => {
      out.value = value
    }, delay)
  })
  return out
}
