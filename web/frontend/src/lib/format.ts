// Number/date formatting. Timestamp unit differs by endpoint (see plan):
//   - account/status/logs → unix SECONDS
//   - /export             → MILLISECONDS
// Use the unit-explicit helpers so the two never get confused.

export function formatNumber(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return '0'
  if (Math.abs(n) >= 1 && Math.floor(n) === n) return Number(n).toLocaleString('en-US')
  return Number(n).toLocaleString('en-US', { maximumFractionDigits: 4 })
}

/** Compact form for KPI tiles: 1.2K, 3.4M, 5.6B. */
export function formatCompact(n: number | null | undefined): string {
  if (n == null || Number.isNaN(n)) return '0'
  const abs = Math.abs(n)
  if (abs >= 1e9) return (n / 1e9).toFixed(1).replace(/\.0$/, '') + 'B'
  if (abs >= 1e6) return (n / 1e6).toFixed(1).replace(/\.0$/, '') + 'M'
  if (abs >= 1e3) return (n / 1e3).toFixed(1).replace(/\.0$/, '') + 'K'
  return formatNumber(n)
}

function toDate(value: number, unit: 'seconds' | 'millis'): Date | null {
  if (!value || value <= 0) return null
  return new Date(unit === 'seconds' ? value * 1000 : value)
}

/** Format a unix-SECONDS timestamp (accounts/status/logs). */
export function formatUnixSeconds(ts: number, opts?: Intl.DateTimeFormatOptions): string {
  const d = toDate(ts, 'seconds')
  if (!d) return '-'
  return d.toLocaleString(undefined, opts ?? { dateStyle: 'medium', timeStyle: 'short' })
}

/** Format a MILLISECONDS timestamp (/export). */
export function formatUnixMillis(ts: number, opts?: Intl.DateTimeFormatOptions): string {
  const d = toDate(ts, 'millis')
  if (!d) return '-'
  return d.toLocaleString(undefined, opts ?? { dateStyle: 'medium', timeStyle: 'short' })
}

/** Clock time only (HH:MM:SS) from unix seconds — used in log rows. */
export function formatClockSeconds(ts: number): string {
  const d = toDate(ts, 'seconds')
  if (!d) return '--:--:--'
  return d.toLocaleTimeString(undefined, { hour12: false })
}

/** "in 3 days" / "5 days ago" style, from unix seconds. Returns '' when 0. */
export function formatRelativeSeconds(ts: number): string {
  const d = toDate(ts, 'seconds')
  if (!d) return ''
  const diffMs = d.getTime() - Date.now()
  const days = Math.round(diffMs / 86_400_000)
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
  if (Math.abs(days) >= 1) return rtf.format(days, 'day')
  const hours = Math.round(diffMs / 3_600_000)
  if (Math.abs(hours) >= 1) return rtf.format(hours, 'hour')
  return rtf.format(Math.round(diffMs / 60_000), 'minute')
}

/** Milliseconds → human duration for request latency: 850ms, 1.2s. */
export function formatDuration(ms: number): string {
  if (!ms || ms < 0) return '0ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/** Uptime seconds → "2d 3h 4m". */
export function formatUptime(seconds: number): string {
  if (!seconds || seconds < 0) return '0m'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d) parts.push(`${d}d`)
  if (h) parts.push(`${h}h`)
  parts.push(`${m}m`)
  return parts.join(' ')
}

export function clampPercent(used: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.max(0, Math.min(100, (used / limit) * 100))
}
