// ApiKeyStatusChart — status breakdown (active/disabled/expired) as a small
// stat row plus a top-keys-by-requests bar. Status colors are the reserved
// status palette (never categorical slots), paired with labels.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts'
import type { ApiKeyView } from '@/types/apikey'
import { seriesColor, chartChrome, STATUS_COLORS } from '@/lib/chartColors'
import { EmptyState } from '@/components/shared/EmptyState'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { formatNumber } from '@/lib/format'

interface ApiKeyStatusChartProps {
  apiKeys: ApiKeyView[]
  loading: boolean
}

export function ApiKeyStatusChart({ apiKeys, loading }: ApiKeyStatusChartProps) {
  const { t } = useTranslation()
  const chrome = chartChrome()

  const { active, disabled, expired, topKeys } = useMemo(() => {
    let active = 0
    let disabled = 0
    let expired = 0
    for (const k of apiKeys) {
      if (k.expired) expired++
      else if (!k.enabled) disabled++
      else active++
    }
    const topKeys = [...apiKeys]
      .sort((a, b) => b.requestsCount - a.requestsCount)
      .slice(0, 8)
      .map((k) => ({ name: k.name || t('apiKeys.unnamed'), requests: k.requestsCount }))
    return { active, disabled, expired, topKeys }
  }, [apiKeys, t])

  if (loading) return <HamsterLoader label={t('detail.loading')} />
  if (apiKeys.length === 0) return <EmptyState message={t('stats.apiKeysNoKeys')} />

  const tiles: Array<{ label: string; value: number; color: string }> = [
    { label: t('apiKeys.filterEnabled'), value: active, color: STATUS_COLORS.good },
    { label: t('apiKeys.filterDisabled'), value: disabled, color: STATUS_COLORS.warning },
    { label: t('apiKeys.filterExpired'), value: expired, color: STATUS_COLORS.critical },
  ]

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-3">
        {tiles.map((tile) => (
          <div key={tile.label} className="rounded-lg border p-3">
            <div className="flex items-center gap-2">
              <span className="size-2.5 rounded-full" style={{ background: tile.color }} />
              <span className="text-xs text-muted-foreground">{tile.label}</span>
            </div>
            <div className="mt-1 text-2xl font-semibold tabular-nums">{formatNumber(tile.value)}</div>
          </div>
        ))}
      </div>

      <div>
        <p className="mb-2 text-xs font-medium text-muted-foreground">{t('stats.apiKeysTopTitle')}</p>
        {topKeys.every((k) => k.requests === 0) ? (
          <EmptyState message={t('stats.apiKeysTopEmpty')} />
        ) : (
          <ResponsiveContainer width="100%" height={Math.max(160, topKeys.length * 32)}>
            <BarChart data={topKeys} layout="vertical" margin={{ left: 8, right: 16 }}>
              <CartesianGrid horizontal={false} stroke={chrome.grid} />
              <XAxis type="number" tick={{ fill: chrome.tick, fontSize: 11 }} stroke={chrome.axis} />
              <YAxis
                type="category"
                dataKey="name"
                width={120}
                tick={{ fill: chrome.tick, fontSize: 11 }}
                stroke={chrome.axis}
              />
              <Tooltip
                cursor={{ fill: chrome.grid, opacity: 0.3 }}
                contentStyle={{
                  background: chrome.surface,
                  border: `1px solid ${chrome.axis}`,
                  borderRadius: 8,
                  color: chrome.text,
                  fontSize: 12,
                }}
              />
              <Bar dataKey="requests" radius={[0, 4, 4, 0]} barSize={16}>
                {topKeys.map((k) => (
                  <Cell key={k.name} fill={seriesColor(0)} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  )
}
