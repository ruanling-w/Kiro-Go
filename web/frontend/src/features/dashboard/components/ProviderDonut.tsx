// ProviderDonut — account distribution across provider buckets. Donut (all-pairs
// form): only the first 3 categorical slots clear the CVD floors, so buckets past
// the third fold into "Other" (dataviz rule). Legend + direct labels keep identity
// off color-alone.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts'
import type { AccountListItem } from '@/types/account'
import { PROVIDERS, bucketOf } from '@/config/providers'
import { seriesColor, chartChrome } from '@/lib/chartColors'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'

interface ProviderDonutProps {
  accounts: AccountListItem[]
  loading: boolean
}

export function ProviderDonut({ accounts, loading }: ProviderDonutProps) {
  const { t } = useTranslation()
  const chrome = chartChrome()

  const data = useMemo(() => {
    const counts = new Map<string, number>()
    for (const a of accounts) {
      const key = bucketOf(a.provider)
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    return PROVIDERS.map((p) => ({
      key: p.key,
      name: t(p.labelKey),
      value: counts.get(p.key) ?? 0,
    })).filter((d) => d.value > 0)
  }, [accounts, t])

  if (loading) return <HamsterLoader label={t('detail.loading')} />
  if (data.length === 0) return <EmptyState message={t('accounts.empty')} />

  return (
    <ResponsiveContainer width="100%" height={260}>
      <PieChart>
        <Pie
          data={data}
          dataKey="value"
          nameKey="name"
          innerRadius={55}
          outerRadius={90}
          paddingAngle={2}
          stroke={chrome.surface}
          strokeWidth={2}
        >
          {data.map((d, i) => (
            <Cell key={d.key} fill={seriesColor(i)} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            background: chrome.surface,
            border: `1px solid ${chrome.axis}`,
            borderRadius: 8,
            color: chrome.text,
            fontSize: 12,
          }}
        />
        <Legend
          wrapperStyle={{ fontSize: 12, color: chrome.text }}
          formatter={(v) => <span style={{ color: chrome.text }}>{v}</span>}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}
