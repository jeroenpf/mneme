// Relative age for list rows: coarse buckets, floored, clamped at 'just now'
// for sub-minute and clock-skewed future timestamps.
const MIN = 60_000
const HOUR = 3_600_000
const DAY = 86_400_000

export function timeAgo(iso: string, now: number): string {
  const delta = now - Date.parse(iso)
  if (delta < MIN) return 'just now'
  if (delta < HOUR) return `${Math.floor(delta / MIN)}m ago`
  if (delta < DAY) return `${Math.floor(delta / HOUR)}h ago`
  if (delta < 7 * DAY) return `${Math.floor(delta / DAY)}d ago`
  if (delta < 30 * DAY) return `${Math.floor(delta / (7 * DAY))}w ago`
  if (delta < 365 * DAY) return `${Math.floor(delta / (30 * DAY))}mo ago`
  return `${Math.floor(delta / (365 * DAY))}y ago`
}
