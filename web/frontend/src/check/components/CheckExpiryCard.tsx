// CheckExpiryCard — focused expiry / lifetime panel.
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { CalendarClock } from 'lucide-react'
import type { CheckKeyResponse } from '@/services/checkKey.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { formatUnixSeconds } from '@/lib/format'
import { cn } from '@/lib/utils'

function formatExpiryDate(expiresAt: number): string {
  if (!expiresAt) return '—'
  // expiresAt is 00:00 of the day after the last valid day — show that day.
  return formatUnixSeconds(expiresAt - 1, { dateStyle: 'full' })
}

export function CheckExpiryCard({ data }: { data: CheckKeyResponse }) {
  const { t } = useTranslation()

  let title: string
  let detail: string
  let badge: ReactNode

  if (data.neverExpires) {
    title = t('check.forever')
    detail = t('check.unlimited')
    badge = (
      <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        {t('check.active')}
      </Badge>
    )
  } else if (data.expired) {
    title = formatExpiryDate(data.expiresAt)
    detail = t('check.expired')
    badge = <Badge variant="destructive">{t('check.expired')}</Badge>
  } else {
    title = formatExpiryDate(data.expiresAt)
    detail = t('check.daysLeft', { n: data.daysRemaining })
    const warn = data.daysRemaining <= 3
    badge = (
      <Badge
        className={cn(
          warn
            ? 'bg-amber-500/15 text-amber-700 dark:text-amber-400'
            : 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400',
        )}
      >
        {detail}
      </Badge>
    )
  }

  return (
    <Card className="h-full">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="flex items-center gap-2">
            <CalendarClock className="size-4 text-muted-foreground" />
            {t('check.kpi.expiry')}
          </CardTitle>
          {badge}
        </div>
      </CardHeader>
      <CardContent className="space-y-2">
        <p className="text-lg font-semibold leading-snug">{title}</p>
        {!data.neverExpires && !data.expired && (
          <p className="text-sm text-muted-foreground">{detail}</p>
        )}
        {data.expired && (
          <p className="text-sm text-destructive">{t('check.expired')}</p>
        )}
        {!data.enabled && (
          <p className="text-sm text-destructive">{t('check.disabled')}</p>
        )}
      </CardContent>
    </Card>
  )
}
