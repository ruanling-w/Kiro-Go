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
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {data.name || data.keyMasked}
          </p>
          <p className="text-xs text-muted-foreground">
            {t('check.autoRefresh')}
            {updatedAt
              ? ` · ${t('check.lastUpdated', {
                  time: formatUnixSeconds(updatedAt, { timeStyle: 'medium' }),
                })}`
              : null}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={refreshing}
          >
            <RefreshCw className={`size-3.5 ${refreshing ? 'animate-spin' : ''}`} />
            {t('check.refresh')}
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={onClear}>
            <LogOut className="size-3.5" />
            {t('check.clear')}
          </Button>
        </div>
      </div>

      <CheckKpiRow data={data} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <CheckQuotaCard data={data} />
        </div>
        <CheckExpiryCard data={data} />
      </div>

      <CheckUsageChart logs={data.logs} />
      <CheckLogsTable logs={data.logs} />
    </div>
  )
}
