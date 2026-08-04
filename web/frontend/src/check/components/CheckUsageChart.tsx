// CheckUsageChart — success vs failed requests over time from check-key logs.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import type { CheckKeyLog } from '@/services/checkKey.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/shared/EmptyState'
import { CHART_SERIES, chartAxis } from '@/lib/chartColors'

interface Bucket {
  label: string
  success: number
  failed: number
}

function groupByHour(logs: CheckKeyLog[]): Bucket[] {
  if (logs.length === 0) return []
  const map = new Map<number, Bucket>()
  for (const log of logs) {
    const hour = Math.floor(log.time / 3600) * 3600
    let b = map.get(hour)
    if (!b) {
      b = {
        label: new Date(hour * 1000).toLocaleString(undefined, {
          month: 'short',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit',
        }),
        success: 0,
        failed: 0,
      }
      map.set(hour, b)
    }
    if (log.status === 'success') b.success += 1
    else b.failed += 1
  }
  return [...map.entries()].sort((a, b) => a[0] - b[0]).map(([, v]) => v)
}

export function CheckUsageChart({ logs }: { logs: CheckKeyLog[] }) {
  const { t } = useTranslation()
  const data = useMemo(() => groupByHour(logs), [logs])
  const axis = chartAxis()

  return (
    <Card className="min-w-0">
      <CardHeader className="min-w-0">
        <CardTitle>{t('check.chart.title')}</CardTitle>
      </CardHeader>
      <CardContent className="min-w-0">
        {data.length === 0 ? (
          <EmptyState message={t('check.logs.empty')} />
        ) : (
          <div className="h-[220px] w-full min-w-0 sm:h-[260px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 8, right: 4, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="checkGradSuccess" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={CHART_SERIES[0]} stopOpacity={0.38} />
                  <stop offset="95%" stopColor={CHART_SERIES[0]} stopOpacity={0} />
                </linearGradient>
                <linearGradient id="checkGradFailed" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={CHART_SERIES[1]} stopOpacity={0.38} />
                  <stop offset="95%" stopColor={CHART_SERIES[1]} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke={axis.grid} vertical={false} />
              <XAxis dataKey="label" stroke={axis.tick} fontSize={11} tickLine={false} minTickGap={28} />
              <YAxis stroke={axis.tick} fontSize={12} tickLine={false} allowDecimals={false} />
              <Tooltip
                contentStyle={axis.tooltip}
                labelStyle={{ color: axis.tooltipLabel }}
              />
              <Area
                type="monotone"
                dataKey="success"
                name={t('check.logs.success')}
                stroke={CHART_SERIES[0]}
                strokeWidth={2}
                fill="url(#checkGradSuccess)"
                activeDot={{ r: 4, strokeWidth: 2, stroke: axis.tooltip.background as string }}
              />
              <Area
                type="monotone"
                dataKey="failed"
                name={t('check.logs.error')}
                stroke={CHART_SERIES[1]}
                strokeWidth={2}
                fill="url(#checkGradFailed)"
                activeDot={{ r: 4, strokeWidth: 2, stroke: axis.tooltip.background as string }}
              />
            </AreaChart>
          </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
