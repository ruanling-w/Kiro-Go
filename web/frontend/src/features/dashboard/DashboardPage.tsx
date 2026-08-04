// Dashboard (Overview + Usage) — KPI row, request/provider charts, api-key
// analytics, and the per-account usage table (merged from the old Usage view).
// Status polls every 10s (useStatus); logs feed the request timeline.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Users, Activity, Coins, Cpu, Gauge } from 'lucide-react'
import { useStatus } from '@/hooks/queries/useStatus'
import { useAccounts } from '@/hooks/queries/useAccounts'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import { useLogs } from '@/hooks/queries/useLogs'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard, type StatTone } from '@/components/shared/StatCard'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { Stagger, StaggerItem } from '@/components/ui/animate/FadeIn'
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

  // Success rate drives the Requests tile's health tone.
  const requestsTone: StatTone =
    !s || s.totalRequests === 0
      ? 'default'
      : successRate >= 98
        ? 'success'
        : successRate >= 90
          ? 'warning'
          : 'danger'

  return (
    <div className="space-y-6">
      <PageHeader title={t('nav.overview')} />

      {status.isPending ? (
        <HamsterLoader label={t('detail.loading')} />
      ) : (
        <Stagger className="grid grid-cols-1 items-stretch gap-4 sm:grid-cols-2 xl:grid-cols-5">
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Users}
              label={t('stats.accounts')}
              count={s?.accounts ?? 0}
              format={formatNumber}
              hint={`${formatNumber(s?.available ?? 0)} · ${t('stats.capacity')}`}
            />
          </StaggerItem>
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Activity}
              label={t('stats.requests')}
              count={s?.totalRequests ?? 0}
              format={formatCompact}
              tone={requestsTone}
            >
              <div className="mt-1 space-y-1 text-xs">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5">
                  <span className="tabular-nums font-medium text-emerald-600 dark:text-emerald-400">
                    {formatCompact(s?.successRequests ?? 0)}{' '}
                    <span className="font-normal text-muted-foreground">
                      {t('stats.success')}
                    </span>
                  </span>
                  <span className="tabular-nums font-medium text-destructive">
                    {formatCompact(s?.failedRequests ?? 0)}{' '}
                    <span className="font-normal text-muted-foreground">
                      {t('stats.failed')}
                    </span>
                  </span>
                </div>
                <span className="text-muted-foreground">
                  {successRate.toFixed(1)}% · {t('stats.reliability')}
                </span>
              </div>
            </StatCard>
          </StaggerItem>
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Cpu}
              label={t('stats.tokens')}
              count={s?.totalTokens ?? 0}
              format={formatCompact}
            />
          </StaggerItem>
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Coins}
              label={t('stats.credits')}
              count={s?.totalCredits ?? 0}
              format={formatCompact}
              hint={s ? formatUptime(s.uptime) : undefined}
            />
          </StaggerItem>
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Gauge}
              label={t('stats.totalRpm')}
              count={s?.totalRpm ?? 0}
              format={formatNumber}
              hint={t('stats.totalRpmHint')}
              live
            />
          </StaggerItem>
        </Stagger>
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
