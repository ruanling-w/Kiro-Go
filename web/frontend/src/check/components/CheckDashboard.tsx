// CheckDashboard — composed mini-dashboard after a successful key lookup.
import { useTranslation } from 'react-i18next'
import { LogOut, RefreshCw } from 'lucide-react'
import type { CheckKeyResponse } from '@/services/checkKey.service'
import { Button } from '@/components/ui/button'
import { formatUnixSeconds } from '@/lib/format'
import { CheckKpiRow } from './CheckKpiRow'
import { CheckQuotaCard } from './CheckQuotaCard'
import { CheckExpiryCard } from './CheckExpiryCard'
import { CheckUsageChart } from './CheckUsageChart'
import { CheckLogsTable } from './CheckLogsTable'

interface CheckDashboardProps {
  data: CheckKeyResponse
  updatedAt?: number
  refreshing?: boolean
  onRefresh: () => void
  onClear: () => void
}

export function CheckDashboard({
  data,
  updatedAt,
  refreshing,
  onRefresh,
  onClear,
}: CheckDashboardProps) {
  const { t } = useTranslation()

  return (
    <div className="min-w-0 space-y-4 sm:space-y-5">
      <div className="flex flex-col gap-3 rounded-xl border bg-card/70 p-3 shadow-sm ring-1 ring-foreground/5 backdrop-blur-sm sm:flex-row sm:flex-wrap sm:items-center sm:justify-between sm:p-4">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold tracking-tight">
            {data.name || data.keyMasked}
          </p>
          <p className="mt-0.5 font-mono text-xs text-muted-foreground">
            {data.keyMasked}
          </p>
          <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
            {t('check.autoRefresh')}
            {updatedAt
              ? ` · ${t('check.lastUpdated', {
                  time: formatUnixSeconds(updatedAt, { timeStyle: 'medium' }),
                })}`
              : null}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="flex-1 sm:flex-none"
            onClick={onRefresh}
            disabled={refreshing}
          >
            <RefreshCw className={`size-3.5 ${refreshing ? 'animate-spin' : ''}`} />
            {t('check.refresh')}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="flex-1 sm:flex-none"
            onClick={onClear}
          >
            <LogOut className="size-3.5" />
            {t('check.clear')}
          </Button>
        </div>
      </div>

      <CheckKpiRow data={data} />

      <div className="grid min-w-0 grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="min-w-0 lg:col-span-2">
          <CheckQuotaCard data={data} />
        </div>
        <div className="min-w-0">
          <CheckExpiryCard data={data} />
        </div>
      </div>

      <CheckUsageChart logs={data.logs} />
      <CheckLogsTable logs={data.logs} />
    </div>
  )
}
