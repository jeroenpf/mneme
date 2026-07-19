import { describe, expect, it, vi } from 'vitest'
import { copyText } from './clipboard'

describe('copyText', () => {
  it('writes the text and resolves true on success', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    expect(await copyText('mneme://document/doc_1')).toBe(true)
    expect(writeText).toHaveBeenCalledWith('mneme://document/doc_1')
  })

  it('resolves false instead of throwing when the clipboard rejects', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('denied'))
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    expect(await copyText('x')).toBe(false)
  })
})
