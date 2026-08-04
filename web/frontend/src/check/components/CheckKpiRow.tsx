// CheckKpiRow — four StatCards: credits, tokens, requests, expiry.
import { useTranslation } from 'react-i18next'
import { Activity, CalendarClock, Coins, Cpu } from 'lucide-react'
import type { CheckKeyResponse } from '@/services/checkKey.service'
import { StatCard, type StatTone } from '@/components/shared/StatCard'
import { Stagger, StaggerItem } from '@/components/ui/animate/FadeIn'
import {
  formatCompact,
  formatNumber,
  formatRelativeSeconds,
  formatUnixSeconds,
} from '@/lib/format'

/** expiresAt is start-of next day; show the inclusive last valid calendar day. */
function formatExpiryDate(expiresAt: number): string {
  if (!expiresAt) return '—'
  return formatUnixSeconds(expiresAt - 1, { dateStyle: 'medium' })
}

function quotaTone(remaining: number, limit: number, unlimited: boolean): StatTone {
  if (unlimited) return 'success'
  if (limit <= 0) return 'default'
  const pct = remaining / limit
  if (remaining <= 0) return 'danger'
  if (pct <= 0.1) return 'danger'
  if (pct <= 0.3) return 'warning'
  return 'success'
}

function expiryTone(data: CheckKeyResponse): StatTone {
  if (data.expired) return 'danger'
  if (data.neverExpires) return 'success'
  if (data.daysRemaining <= 3) return 'warning'
  return 'success'
}

export function CheckKpiRow({ data }: { data: CheckKeyResponse }) {
  const { t } = useTranslation()

  const creditsValue = data.creditUnlimited
    ? '∞'
    : formatCompact(data.creditsRemaining)
  const creditsHint = data.creditUnlimited
    ? t('check.unlimited')
    : t('check.kpi.usedOf', {
        used: formatNumber(data.creditsUsed),
        limit: formatNumber(data.creditLimit),
      })

  const tokensValue = data.tokenUnlimited ? '∞' : formatCompact(data.tokensRemaining)
  const tokensHint = data.tokenUnlimited
    ? t('check.unlimited')
    : t('check.kpi.usedOf', {
        used: formatNumber(data.tokensUsed),
        limit: formatNumber(data.tokenLimit),
      })

  const lastUsedHint = data.lastUsedAt
    ? t('check.kpi.lastUsed', { time: formatRelativeSeconds(data.lastUsedAt) })
    : t('check.kpi.neverUsed')

  let expiryValue: string
  let expiryHint: string
  if (data.neverExpires) {
    expiryValue = t('check.forever')
    expiryHint = t('check.unlimited')
  } else if (data.expired) {
    expiryValue = formatExpiryDate(data.expiresAt)
    expiryHint = t('check.expired')
  } else {
    expiryValue = formatExpiryDate(data.expiresAt)
    expiryHint = t('check.daysLeft', { n: data.daysRemaining })
  }

  return (
    <Stagger className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StaggerItem>
        <StatCard
          icon={Coins}
          label={t('check.kpi.credits')}
          value={creditsValue}
          hint={creditsHint}
          tone={quotaTone(data.creditsRemaining, data.creditLimit, data.creditUnlimited)}
        />
      </StaggerItem>
      <StaggerItem>
        <StatCard
          icon={Cpu}
          label={t('check.kpi.tokens')}
          value={tokensValue}
          hint={tokensHint}
          tone={quotaTone(data.tokensRemaining, data.tokenLimit, data.tokenUnlimited)}
        />
      </StaggerItem>
      <StaggerItem>
        <StatCard
          icon={Activity}
          label={t('check.kpi.requests')}
          count={data.requestsCount}
          format={formatCompact}
          hint={lastUsedHint}
        />
      </StaggerItem>
      <StaggerItem>
        <StatCard
          icon={CalendarClock}
          label={t('check.kpi.expiry')}
          value={expiryValue}
          hint={expiryHint}
          tone={expiryTone(data)}
        />
      </StaggerItem>
    </Stagger>
  )
}
