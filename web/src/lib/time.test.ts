import { describe, expect, it } from 'vitest'
import { timeAgo } from './time'

const NOW = Date.parse('2026-07-28T12:00:00Z')

function isoBefore(ms: number): string {
  return new Date(NOW - ms).toISOString()
}

const MIN = 60_000
const HOUR = 3_600_000
const DAY = 86_400_000

describe('timeAgo', () => {
  it('clamps sub-minute ages to "just now"', () => {
    expect(timeAgo(isoBefore(30_000), NOW)).toBe('just now')
  })

  it('clamps future timestamps (clock skew) to "just now"', () => {
    expect(timeAgo(isoBefore(-HOUR), NOW)).toBe('just now')
  })

  it('floors minutes', () => {
    expect(timeAgo(isoBefore(MIN), NOW)).toBe('1m ago')
    expect(timeAgo(isoBefore(59 * MIN), NOW)).toBe('59m ago')
  })

  it('floors hours', () => {
    expect(timeAgo(isoBefore(90 * MIN), NOW)).toBe('1h ago')
    expect(timeAgo(isoBefore(26 * HOUR), NOW)).toBe('1d ago')
  })

  it('floors days up to a week', () => {
    expect(timeAgo(isoBefore(6 * DAY), NOW)).toBe('6d ago')
    expect(timeAgo(isoBefore(7 * DAY), NOW)).toBe('1w ago')
  })

  it('floors weeks up to a month', () => {
    expect(timeAgo(isoBefore(29 * DAY), NOW)).toBe('4w ago')
    expect(timeAgo(isoBefore(30 * DAY), NOW)).toBe('1mo ago')
  })

  it('floors months up to a year', () => {
    expect(timeAgo(isoBefore(364 * DAY), NOW)).toBe('12mo ago')
    expect(timeAgo(isoBefore(365 * DAY), NOW)).toBe('1y ago')
  })

  it('floors years beyond', () => {
    expect(timeAgo(isoBefore(800 * DAY), NOW)).toBe('2y ago')
  })
})
