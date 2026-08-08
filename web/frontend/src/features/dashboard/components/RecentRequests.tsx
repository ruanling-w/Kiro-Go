// RecentRequests — compact table: Model | In/Out | When (last 15 entries from SSE stream).
// Model column reuses the shared BrandChip so colors/logos match the Logs table.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { RequestLog } from '@/types/log'
import { BrandChip, brandFor } from '@/components/shared/ModelBrand'
import { formatCompact } from '@/lib/format'
import { cn } from '@/lib/utils'

interface Props {
  logs: RequestLog[]
}

function relativeTime(unixSec: number, t: (k: string, v?: Record<string, unknown>) => string): string {
  const diff = Math.floor(Date.now() / 1000) - unixSec
  if (diff < 60) return t('recent.justNow')
  if (diff < 3600) return t('recent.minutesAgo', { 0: Math.floor(diff / 60) })
  if (diff < 86400) return t('recent.hoursAgo', { 0: Math.floor(diff / 3600) })
  return t('recent.daysAgo', { 0: Math.floor(diff / 86400) })
}

export function RecentRequests({ logs }: Props) {
  const { t } = useTranslation()
  const recent = useMemo(() => logs.slice(0, 15), [logs])

  if (recent.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-muted-foreground">{t('recent.empty')}</p>
    )
  }

  return (
    <div className="w-full overflow-hidden">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b text-muted-foreground">
            <th className="pb-2 text-left font-medium">{t('recent.model')}</th>
            <th className="pb-2 text-right font-medium">{t('recent.inOut')}</th>
            <th className="pb-2 text-right font-medium">{t('recent.when')}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border/50">
          {recent.map((log, idx) => {
            const isSuccess = log.status === 'success'
            const inp = log.inputTokens ?? 0
            const out = log.outputTokens ?? 0
            const brand = brandFor(log.provider, log.model, t)
            const label = log.model || log.endpoint
            return (
              <tr key={idx} className="hover:bg-muted/30 transition-colors">
                <td className="py-2 pr-2">
                  <div className="flex items-center gap-1.5 min-w-0">
                    <span
                      className={cn(
                        'size-1.5 shrink-0 rounded-full',
                        isSuccess ? 'bg-emerald-500' : 'bg-destructive',
                      )}
                    />
                    {brand ? (
                      <BrandChip brand={brand} text={label} title={label} />
                    ) : (
                      <span className="truncate font-medium text-foreground">{label}</span>
                    )}
                  </div>
                </td>
                <td className="py-2 text-right tabular-nums whitespace-nowrap">
                  <span className="text-emerald-600 dark:text-emerald-400">
                    {formatCompact(inp)}↑
                  </span>{' '}
                  <span className="text-destructive">
                    {formatCompact(out)}↓
                  </span>
                </td>
                <td className="py-2 pl-3 text-right text-muted-foreground whitespace-nowrap">
                  {relativeTime(log.time, t)}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
