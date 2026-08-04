// CheckQuotaCard — usage bars + key meta (masked key, name, timestamps).
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { CheckKeyResponse } from '@/services/checkKey.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { UsageBar } from '@/components/shared/UsageBar'
import { formatNumber, formatUnixSeconds } from '@/lib/format'

function statusBadge(data: CheckKeyResponse, t: (k: string) => string) {
  if (!data.enabled) {
    return <Badge variant="destructive">{t('check.disabled')}</Badge>
  }
  if (data.expired) {
    return <Badge variant="destructive">{t('check.expired')}</Badge>
  }
  return (
    <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
      {t('check.active')}
    </Badge>
  )
}

function MetaRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 items-start justify-between gap-3 border-b border-border/60 py-2 last:border-0 sm:items-center">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span className="min-w-0 break-words text-right font-medium">{children}</span>
    </div>
  )
}

export function CheckQuotaCard({ data }: { data: CheckKeyResponse }) {
  const { t } = useTranslation()

  return (
    <Card className="h-full min-w-0">
      <CardHeader className="min-w-0 pb-2">
        <div className="flex min-w-0 items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <CardTitle className="break-words">{data.name || t('check.meta.title')}</CardTitle>
            <p className="mt-1 break-all font-mono text-xs text-muted-foreground">
              {data.keyMasked}
            </p>
          </div>
          <div className="shrink-0">{statusBadge(data, t)}</div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-3">
          {data.creditUnlimited ? (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('check.quota.credits')}</span>
              <Badge variant="secondary">{t('check.unlimited')}</Badge>
            </div>
          ) : (
            <UsageBar
              label={t('check.quota.credits')}
              used={data.creditsUsed}
              limit={data.creditLimit}
              format={formatNumber}
            />
          )}
          {data.tokenUnlimited ? (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('check.quota.tokens')}</span>
              <Badge variant="secondary">{t('check.unlimited')}</Badge>
            </div>
          ) : (
            <UsageBar
              label={t('check.quota.tokens')}
              used={data.tokensUsed}
              limit={data.tokenLimit}
              format={formatNumber}
            />
          )}
        </div>

        <div className="text-sm">
          <MetaRow label={t('check.meta.requests')}>{formatNumber(data.requestsCount)}</MetaRow>
          <MetaRow label={t('check.meta.created')}>
            {formatUnixSeconds(data.createdAt, { dateStyle: 'medium', timeStyle: 'short' })}
          </MetaRow>
          <MetaRow label={t('check.meta.lastUsed')}>
            {data.lastUsedAt
              ? formatUnixSeconds(data.lastUsedAt, { dateStyle: 'medium', timeStyle: 'short' })
              : '—'}
          </MetaRow>
        </div>
      </CardContent>
    </Card>
  )
}
