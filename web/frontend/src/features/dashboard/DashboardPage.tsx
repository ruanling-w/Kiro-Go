// Dashboard (Overview + Usage) — KPI row, request/provider charts, api-key
// analytics, and the per-account usage table (merged from the old Usage view).
// Status polls every 10s (useStatus); logs feed the request timeline.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Activity, Coins, Cpu } from 'lucide-react'
import { useStatus } from '@/hooks/queries/useStatus'
import { useAccounts } from '@/hooks/queries/useAccounts'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import { useLogs } from '@/hooks/queries/useLogs'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard } from '@/components/shared/StatCard'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatCompact, formatNumber, formatUptime } from '@/lib/format'
import { RequestsChart } from './components/RequestsChart'
import { ProviderDonut } from './components/ProviderDonut'
import { ApiKeyStatusChart } from './components/ApiKeyStatusChart'
import { UsageTable } from './components/UsageTable'

export default function DashboardPage() {
  const { t } = useTranslation()
  const status = useStatus()
  const accounts = useAccounts()
  const apiKeys = useApiKeys()
  const logs = useLogs(false)

  const s = status.data

  const successRate = useMemo(() => {
    if (!s || s.totalRequests === 0) return 0
    return (s.successRequests / s.totalRequests) * 100
  }, [s])

  return (
    <div className="space-y-6">
      <PageHeader title={t('nav.overview')} />

      {status.isPending ? (
        <HamsterLoader label={t('detail.loading')} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            icon={Users}
            label={t('stats.accounts')}
            value={formatNumber(s?.accounts ?? 0)}
            hint={`${formatNumber(s?.available ?? 0)} · ${t('stats.capacity')}`}
          />
          <StatCard
            icon={Activity}
            label={t('stats.requests')}
            value={formatCompact(s?.totalRequests ?? 0)}
            hint={`${successRate.toFixed(1)}% · ${t('stats.reliability')}`}
          />
          <StatCard
            icon={Cpu}
            label={t('stats.tokens')}
            value={formatCompact(s?.totalTokens ?? 0)}
          />
          <StatCard
            icon={Coins}
            label={t('stats.credits')}
            value={formatCompact(s?.totalCredits ?? 0)}
            hint={s ? formatUptime(s.uptime) : undefined}
          />
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('stats.traffic')}</CardTitle>
          </CardHeader>
          <CardContent>
            <RequestsChart logs={logs.data ?? []} loading={logs.isPending} />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t('nav.providers')}</CardTitle>
          </CardHeader>
          <CardContent>
            <ProviderDonut accounts={accounts.data ?? []} loading={accounts.isPending} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('stats.apiKeysStatusTitle')}</CardTitle>
        </CardHeader>
        <CardContent>
          <ApiKeyStatusChart apiKeys={apiKeys.data ?? []} loading={apiKeys.isPending} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('nav.usage')}</CardTitle>
        </CardHeader>
        <CardContent>
          <UsageTable accounts={accounts.data ?? []} loading={accounts.isPending} />
        </CardContent>
      </Card>
    </div>
  )
}
