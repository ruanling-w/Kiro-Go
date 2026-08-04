// CheckLogsTable — filterable usage history for one API key.
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { CheckKeyLog } from '@/services/checkKey.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
            className="sm:max-w-xs"
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
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('check.logs.time')}</TableHead>
                <TableHead>{t('check.logs.endpoint')}</TableHead>
                <TableHead>{t('check.logs.model')}</TableHead>
                <TableHead>{t('check.logs.status')}</TableHead>
                <TableHead className="text-right">{t('check.logs.tokens')}</TableHead>
                <TableHead className="text-right">{t('check.logs.credits')}</TableHead>
                <TableHead className="text-right">{t('check.logs.duration')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((log, i) => {
                const ok = log.status === 'success'
                return (
                  <TableRow key={`${log.time}-${log.model}-${i}`}>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {formatUnixSeconds(log.time, {
                        dateStyle: 'short',
                        timeStyle: 'medium',
                      })}
                    </TableCell>
                    <TableCell className="max-w-[8rem] truncate font-mono text-xs">
                      {log.endpoint || '—'}
                    </TableCell>
                    <TableCell className="max-w-[10rem] truncate">{log.model || '—'}</TableCell>
                    <TableCell>
                      {ok ? (
                        <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
                          {t('check.logs.success')}
                        </Badge>
                      ) : (
                        <Badge variant="destructive">
                          {log.errorType || t('check.logs.error')}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {formatNumber(log.tokens)}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums">
                      {formatNumber(log.credits)}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                      {formatDuration(log.duration)}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
