// ApiKeyStatusChart — status breakdown + top keys by lifetime requests and
// live RPM (60s window from the gateway).
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

  const { active, disabled, expired, topKeys, topRpm } = useMemo(() => {
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
    const topRpm = [...apiKeys]
      .sort((a, b) => (b.rpm || 0) - (a.rpm || 0))
      .slice(0, 8)
      .map((k) => ({
        name: k.name || t('apiKeys.unnamed'),
        rpm: k.rpm || 0,
        limit: k.rpmLimit || 0,
      }))
    return { active, disabled, expired, topKeys, topRpm }
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

      <div className="grid gap-6 lg:grid-cols-2">
        <div>
          <p className="mb-2 text-xs font-medium text-muted-foreground">
            {t('stats.apiKeysTopTitle')}
          </p>
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

        <div>
          <p className="mb-2 text-xs font-medium text-muted-foreground">
            {t('stats.apiKeysTopRpmTitle')}
          </p>
          {topRpm.every((k) => k.rpm === 0) ? (
            <EmptyState message={t('stats.apiKeysTopRpmEmpty')} />
          ) : (
            <ResponsiveContainer width="100%" height={Math.max(160, topRpm.length * 32)}>
              <BarChart data={topRpm} layout="vertical" margin={{ left: 8, right: 16 }}>
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
                  formatter={(value, _name, item) => {
                    const lim = (item?.payload as { limit?: number } | undefined)?.limit ?? 0
                    const v = typeof value === 'number' ? value : Number(value) || 0
                    return lim > 0 ? `${formatNumber(v)} / ${formatNumber(lim)}` : formatNumber(v)
                  }}
                  contentStyle={{
                    background: chrome.surface,
                    border: `1px solid ${chrome.axis}`,
                    borderRadius: 8,
                    color: chrome.text,
                    fontSize: 12,
                  }}
                />
                <Bar dataKey="rpm" radius={[0, 4, 4, 0]} barSize={16}>
                  {topRpm.map((k) => (
                    <Cell
                      key={k.name}
                      fill={
                        k.limit > 0 && k.rpm >= k.limit
                          ? STATUS_COLORS.critical
                          : k.limit > 0 && k.rpm >= k.limit * 0.8
                            ? STATUS_COLORS.warning
                            : seriesColor(1)
                      }
                    />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  )
}
