// Dashboard (Overview + Usage) — KPI row, topology graph, recent requests,
// provider charts, api-key analytics, and the per-account usage table.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Activity, Coins, Cpu, Gauge, Database } from 'lucide-react'
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
import { ProxyTopologyGraph } from './components/ProxyTopologyGraph'
import { RecentRequests } from './components/RecentRequests'

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
        <Stagger className="grid grid-cols-1 items-stretch gap-4 sm:grid-cols-2 xl:grid-cols-6">
          {/* 1 — Requests with success/failed breakdown */}
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
                    <span className="font-normal text-muted-foreground">{t('stats.success')}</span>
                  </span>
                  <span className="tabular-nums font-medium text-destructive">
                    {formatCompact(s?.failedRequests ?? 0)}{' '}
                    <span className="font-normal text-muted-foreground">{t('stats.failed')}</span>
                  </span>
                </div>
                <span className="text-muted-foreground">
                  {successRate.toFixed(1)}% · {t('stats.reliability')}
                </span>
              </div>
            </StatCard>
          </StaggerItem>

          {/* 3 — Tokens with input/output breakdown */}
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Cpu}
              label={t('stats.tokens')}
              count={s?.totalTokens ?? 0}
              format={formatCompact}
            >
              <div className="mt-1 space-y-1 text-xs">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5">
                  <span className="tabular-nums font-medium text-sky-600 dark:text-sky-400">
                    {formatCompact(s?.totalInputTokens ?? 0)}{' '}
                    <span className="font-normal text-muted-foreground">in</span>
                  </span>
                  <span className="tabular-nums font-medium text-emerald-600 dark:text-emerald-400">
                    {formatCompact(s?.totalOutputTokens ?? 0)}{' '}
                    <span className="font-normal text-muted-foreground">out</span>
                  </span>
                  {(s?.totalLegacyTokens ?? 0) > 0 && (
                    <span
                      className="tabular-nums font-medium text-amber-600 dark:text-amber-400"
                      title="Historical tokens recorded before input/output breakdown was available"
                    >
                      {formatCompact(s?.totalLegacyTokens ?? 0)}{' '}
                      <span className="font-normal text-muted-foreground">legacy</span>
                    </span>
                  )}
                </div>
              </div>
            </StatCard>
          </StaggerItem>

          {/* 4 — Provider prompt cache usage */}
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Database}
              label={t('stats.cacheReadTokens')}
              count={s?.totalCacheReadTokens ?? s?.totalCacheTokens ?? 0}
              format={formatCompact}
            >
              <div className="mt-1 space-y-1 text-xs text-muted-foreground">
                <div>
                  {formatCompact(s?.totalCacheCreationTokens ?? 0)} {t('stats.cacheWriteTokens')}
                </div>
                <div>
                  {formatCompact(s?.totalResponseCacheHits ?? 0)} {t('stats.responseCacheHits')}
                </div>
              </div>
            </StatCard>
          </StaggerItem>

          {/* 5 — Credits */}
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

          {/* 6 — Est. Cost */}
          <StaggerItem className="h-full min-h-0">
            <StatCard
              className="h-full"
              icon={Coins}
              label={t('stats.estCost')}
              count={s?.totalCredits ?? 0}
              format={(n) => `~$${n.toFixed(2)}`}
              tone="warning"
              hint={t('stats.estCostHint')}
            />
          </StaggerItem>

          {/* 7 — RPM live */}
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

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <Card className="flex flex-col lg:col-span-3">
          <CardContent className="flex-1 p-0">
            <ProxyTopologyGraph accounts={accounts.data ?? []} stats={s} />
          </CardContent>
        </Card>
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('stats.recentRequests')}</CardTitle>
          </CardHeader>
          <CardContent>
            <RecentRequests logs={logs.data ?? []} />
          </CardContent>
        </Card>
      </div>

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
