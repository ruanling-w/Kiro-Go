// CheckLogsTable — filterable usage history for one API key. Uses shared
// DataTable so mobile gets the same card stack as the admin tables.
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { CheckKeyLog } from '@/services/checkKey.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { EmptyState } from '@/components/shared/EmptyState'
import { formatDuration, formatNumber, formatUnixSeconds } from '@/lib/format'
import { cn } from '@/lib/utils'

type StatusFilter = 'all' | 'success' | 'error'

export function CheckLogsTable({ logs }: { logs: CheckKeyLog[] }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<StatusFilter>('all')
  const [q, setQ] = useState('')

  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase()
    return logs.filter((log) => {
      if (status === 'success' && log.status !== 'success') return false
      if (status === 'error' && log.status === 'success') return false
      if (!needle) return true
      const hay = `${log.model ?? ''} ${log.endpoint ?? ''} ${log.errorType ?? ''}`.toLowerCase()
      return hay.includes(needle)
    })
  }, [logs, status, q])

  const filters: { id: StatusFilter; label: string }[] = [
    { id: 'all', label: t('check.logs.filterAll') },
    { id: 'success', label: t('check.logs.filterSuccess') },
    { id: 'error', label: t('check.logs.filterError') },
  ]

  const columns = useMemo<Column<CheckKeyLog>[]>(
    () => [
      {
        id: 'time',
        header: t('check.logs.time'),
        mobileRole: 'primary',
        sortValue: (log) => log.time,
        cell: (log) => (
          <span className="font-mono text-xs text-muted-foreground">
            {formatUnixSeconds(log.time, {
              dateStyle: 'short',
              timeStyle: 'medium',
            })}
          </span>
        ),
      },
      {
        id: 'endpoint',
        header: t('check.logs.endpoint'),
        mobileRole: 'meta',
        cell: (log) => (
          <span className="max-w-[12rem] truncate font-mono text-xs">
            {log.endpoint || '—'}
          </span>
        ),
        sortValue: (log) => log.endpoint || '',
      },
      {
        id: 'model',
        header: t('check.logs.model'),
        mobileRole: 'secondary',
        cell: (log) => (
          <span className="max-w-[14rem] truncate text-sm">{log.model || '—'}</span>
        ),
        sortValue: (log) => log.model || '',
      },
      {
        id: 'status',
        header: t('check.logs.status'),
        mobileRole: 'meta',
        cell: (log) => {
          const ok = log.status === 'success'
          return ok ? (
            <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
              {t('check.logs.success')}
            </Badge>
          ) : (
            <Badge variant="destructive">
              {log.errorType || t('check.logs.error')}
            </Badge>
          )
        },
        sortValue: (log) => (log.status === 'success' ? 1 : 0),
      },
      {
        id: 'tokens',
        header: t('check.logs.tokens'),
        mobileRole: 'meta',
        align: 'right',
        cell: (log) => (
          <span className="font-mono tabular-nums">{formatNumber(log.tokens)}</span>
        ),
        sortValue: (log) => log.tokens,
      },
      {
        id: 'credits',
        header: t('check.logs.credits'),
        mobileRole: 'meta',
        align: 'right',
        cell: (log) => (
          <span className="font-mono tabular-nums">{formatNumber(log.credits)}</span>
        ),
        sortValue: (log) => log.credits,
      },
      {
        id: 'duration',
        header: t('check.logs.duration'),
        mobileRole: 'meta',
        align: 'right',
        cell: (log) => (
          <span className="font-mono tabular-nums text-muted-foreground">
            {formatDuration(log.duration)}
          </span>
        ),
        sortValue: (log) => log.duration,
      },
    ],
    [t],
  )

  return (
    <Card>
      <CardHeader className="gap-3 space-y-0">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle>{t('check.logs.title')}</CardTitle>
          <span className="text-xs text-muted-foreground">
            {t('check.logs.count', { n: logs.length })}
          </span>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t('check.logs.search')}
            className="min-w-0 sm:max-w-xs"
          />
          <div className="flex flex-wrap gap-1.5">
            {filters.map((f) => (
              <button
                key={f.id}
                type="button"
                onClick={() => setStatus(f.id)}
                className={cn(
                  'rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  status === f.id
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80',
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {logs.length === 0 ? (
          <EmptyState message={t('check.logs.empty')} />
        ) : filtered.length === 0 ? (
          <EmptyState message={t('check.logs.noMatch')} />
        ) : (
          <DataTable
            rows={filtered}
            columns={columns}
            rowKey={(log) =>
              `${log.time}-${log.model ?? ''}-${log.endpoint ?? ''}-${log.status}-${log.tokens}-${log.duration}`
            }
            initialSort={{ id: 'time', dir: 'desc' }}
            pageSize={20}
            emptyMessage={t('check.logs.noMatch')}
          />
        )}
      </CardContent>
    </Card>
  )
}
