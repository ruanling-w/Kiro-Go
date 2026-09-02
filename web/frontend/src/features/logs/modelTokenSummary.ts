import type { LiveLog } from '@/hooks/queries/useLogStream'

export interface ModelTokenSummaryRow {
  model: string
  totalTokens: number
  cacheReadTokens: number
  cacheCreationTokens: number
  requests: number
  avgTokens: number
}

export function toUnixDayStart(date: string): number | null {
  if (!date) return null
  const d = new Date(`${date}T00:00:00`)
  if (Number.isNaN(d.getTime())) return null
  return Math.floor(d.getTime() / 1000)
}

export function toUnixDayEnd(date: string): number | null {
  if (!date) return null
  const d = new Date(`${date}T23:59:59.999`)
  if (Number.isNaN(d.getTime())) return null
  return Math.floor(d.getTime() / 1000)
}

export function requestTokenTotal(log: Pick<LiveLog, 'tokens' | 'inputTokens' | 'outputTokens' | 'cacheReadTokens' | 'cacheCreationTokens'>): number {
  if (typeof log.tokens === 'number' && log.tokens > 0) {
    return log.tokens
  }
  return (log.inputTokens ?? 0) + (log.outputTokens ?? 0) + (log.cacheReadTokens ?? 0) + (log.cacheCreationTokens ?? 0)
}

export type ModelSummarySort =
  | 'total-desc'
  | 'total-asc'
  | 'avg-desc'
  | 'avg-asc'
  | 'requests-desc'
  | 'model-asc'

export function summarizeModelTokens(
  logs: LiveLog[],
  startDate?: string,
  endDate?: string,
): ModelTokenSummaryRow[] {
  const start = toUnixDayStart(startDate ?? '')
  const end = toUnixDayEnd(endDate ?? '')

  const map = new Map<string, {
    totalTokens: number
    cacheReadTokens: number
    cacheCreationTokens: number
    requests: number
  }>()

  for (const log of logs) {
    if (log.status !== 'success') continue
    if (start !== null && log.time < start) continue
    if (end !== null && log.time > end) continue

    const model = (log.model ?? '').trim()
    if (!model) continue

    const total = requestTokenTotal(log)
    if (total <= 0) continue

    const current = map.get(model)
    const cacheRead = log.cacheReadTokens ?? 0
    const cacheWrite = log.cacheCreationTokens ?? 0

    if (current) {
      current.totalTokens += total
      current.cacheReadTokens += cacheRead
      current.cacheCreationTokens += cacheWrite
      current.requests += 1
    } else {
      map.set(model, {
        totalTokens: total,
        cacheReadTokens: cacheRead,
        cacheCreationTokens: cacheWrite,
        requests: 1,
      })
    }
  }

  return Array.from(map.entries())
    .map(([model, value]) => ({
      model,
      totalTokens: value.totalTokens,
      cacheReadTokens: value.cacheReadTokens,
      cacheCreationTokens: value.cacheCreationTokens,
      requests: value.requests,
      avgTokens: Math.round(value.totalTokens / value.requests),
    }))
}

export function sortModelTokenSummary(
  rows: ModelTokenSummaryRow[],
  sort: ModelSummarySort,
): ModelTokenSummaryRow[] {
  const next = [...rows]
  next.sort((a, b) => {
    switch (sort) {
      case 'total-asc':
        return a.totalTokens - b.totalTokens
      case 'avg-desc':
        return b.avgTokens - a.avgTokens
      case 'avg-asc':
        return a.avgTokens - b.avgTokens
      case 'requests-desc':
        return b.requests - a.requests
      case 'model-asc':
        return a.model.localeCompare(b.model)
      case 'total-desc':
      default:
        return b.totalTokens - a.totalTokens
    }
  })
  return next
}
