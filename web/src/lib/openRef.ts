import type { Router } from 'vue-router'
import { parseRef, routeForRef } from './mnemeRef'

// tryOpenRef inspects a search-box value: if it is a mneme:// reference or a
// bare public id, it navigates straight to the target and returns true; the
// caller falls back to full-text search only when this returns false.
export function tryOpenRef(router: Router, input: string): boolean {
  const ref = parseRef(input)
  if (!ref) return false
  router.push(routeForRef(ref))
  return true
}
