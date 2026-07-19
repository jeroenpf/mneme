// clipboard.ts — copy text to the clipboard, resolving true on success and
// false (rather than throwing) on failure so callers can surface a gentle
// "copy failed" toast instead of an uncaught rejection.
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
