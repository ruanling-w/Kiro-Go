// RequestsChart — success vs failed requests over time, grouped from the logs
// by hour bucket (client-side). Area chart, two categorical hues (blue/orange).
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
import type { RequestLog } from '@/types/log'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { CHART_SERIES, chartAxis } from '@/lib/chartColors'

interface Bucket {
  label: string
  success: number
  failed: number
}

function groupByHour(logs: RequestLog[]): Bucket[] {
  if (logs.length === 0) return []
  const map = new Map<number, Bucket>()
  for (const log of logs) {
    const hour = Math.floor(log.time / 3600) * 3600
    let b = map.get(hour)
    if (!b) {
      b = {
        label: new Date(hour * 1000).toLocaleTimeString(undefined, {
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

export function RequestsChart({ logs, loading }: { logs: RequestLog[]; loading: boolean }) {
  const { t } = useTranslation()
  const data = useMemo(() => groupByHour(logs), [logs])

  if (loading) return <HamsterLoader label={t('detail.loading')} />
  if (data.length === 0) return <EmptyState message={t('logs.empty')} />

  const axis = chartAxis()

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data} margin={{ top: 8, right: 8, left: -16, bottom: 0 }}>
        <defs>
          <linearGradient id="gradSuccess" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={CHART_SERIES[0]} stopOpacity={0.3} />
            <stop offset="95%" stopColor={CHART_SERIES[0]} stopOpacity={0} />
          </linearGradient>
          <linearGradient id="gradFailed" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={CHART_SERIES[1]} stopOpacity={0.3} />
            <stop offset="95%" stopColor={CHART_SERIES[1]} stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke={axis.grid} vertical={false} />
        <XAxis dataKey="label" stroke={axis.tick} fontSize={12} tickLine={false} />
        <YAxis stroke={axis.tick} fontSize={12} tickLine={false} allowDecimals={false} />
        <Tooltip
          contentStyle={axis.tooltip}
          labelStyle={{ color: axis.tooltipLabel }}
        />
        <Area
          type="monotone"
          dataKey="success"
          name={t('logs.success')}
          stroke={CHART_SERIES[0]}
          strokeWidth={2}
          fill="url(#gradSuccess)"
        />
        <Area
          type="monotone"
          dataKey="failed"
          name={t('logs.errors')}
          stroke={CHART_SERIES[1]}
          strokeWidth={2}
          fill="url(#gradFailed)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
