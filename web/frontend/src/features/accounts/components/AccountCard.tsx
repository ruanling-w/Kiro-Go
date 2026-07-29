// AccountCard — one account's at-a-glance card: identity + status badges, usage
// bars (main + trial quota), 4 runtime stats, and per-card actions (test,
// refresh, detail, delete). Email is masked when privacy mode is on. Selection
// checkbox drives the batch bar. Provider-specific quota panels are shown when
// the provider exposes them.
import { useTranslation } from 'react-i18next'
import { RefreshCw, FlaskConical, Settings2, Trash2, CheckSquare, Square } from 'lucide-react'
import type { AccountListItem } from '@/types/account'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { StatusBadge, type StatusTone } from '@/components/shared/StatusBadge'
import { UsageBar } from '@/components/shared/UsageBar'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { useUiStore } from '@/stores/uiStore'
import { maskEmail } from '@/lib/mask'
import { formatNumber, formatRelativeSeconds } from '@/lib/format'
import { bucketOf } from '@/config/providers'
import { cn } from '@/lib/utils'

interface Props {
  account: AccountListItem
  onTest: (a: AccountListItem) => void
  onRefresh: (a: AccountListItem) => void
  onDetail: (a: AccountListItem) => void
  onDelete: (a: AccountListItem) => void
  refreshing?: boolean
  testing?: boolean
}

function statusTone(a: AccountListItem): { tone: StatusTone; key: string } {
  if (a.banStatus && a.banStatus !== 'none' && a.banStatus !== '') {
    return { tone: 'danger', key: 'accounts.banned' }
  }
  if (!a.enabled) return { tone: 'neutral', key: 'accounts.disabled' }
  if (!a.hasToken) return { tone: 'warning', key: 'accounts.noToken' }
  return { tone: 'success', key: 'accounts.enabled' }
}

export function AccountCard({
  account: a,
  onTest,
  onRefresh,
  onDetail,
  onDelete,
  refreshing,
  testing,
}: Props) {
  const { t } = useTranslation()
  const privacy = useUiStore((s) => s.privacyMode)
  const selected = useUiStore((s) => s.selectedAccounts.has(a.id))
  const toggleSelected = useUiStore((s) => s.toggleAccountSelected)

  const status = statusTone(a)
  const bucket = bucketOf(a.provider)
  const email = privacy ? maskEmail(a.email || a.id) : a.email || a.id

  const SelIcon = selected ? CheckSquare : Square

  return (
    <Card
      interactive
      className={cn(
        'flex flex-col gap-3 p-4',
        selected && 'ring-2 ring-primary/60 shadow-[0_8px_24px_-14px_var(--glow)]',
      )}
    >
      <div className="flex items-start gap-3">
        <button
          type="button"
          onClick={() => toggleSelected(a.id)}
          className={cn(
            'mt-0.5 transition-colors hover:text-foreground',
            selected ? 'text-primary' : 'text-muted-foreground',
          )}
          aria-label={t('accounts.selectAccount')}
        >
          <SelIcon className="size-4.5" />
        </button>
        <ProviderIcon provider={bucket} className="mt-0.5" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="truncate font-medium" title={email}>
              {a.nickname || email}
            </p>
            {a.weight !== 1 && a.weight > 0 && (
              <span className="shrink-0 rounded bg-muted px-1.5 text-xs text-muted-foreground">
                {t('accounts.weightShort')}{a.weight}
              </span>
            )}
          </div>
          {a.nickname && (
            <p className="truncate text-xs text-muted-foreground" title={email}>
              {email}
            </p>
          )}
        </div>
        <StatusBadge tone={status.tone}>{t(status.key)}</StatusBadge>
      </div>

      {a.subscriptionTitle && (
        <p className="text-xs text-muted-foreground">{a.subscriptionTitle}</p>
      )}

      {a.usageLimit > 0 && (
        <UsageBar
          label={t('accounts.mainQuota')}
          used={a.usageCurent}
          limit={a.usageLimit}
        />
      )}
      {a.trialUsageLimit > 0 && (
        <UsageBar
          label={t('accounts.trialQuota')}
          used={a.trialUsageCurent}
          limit={a.trialUsageLimit}
        />
      )}

      <div className="grid grid-cols-4 gap-2 rounded-lg bg-muted/50 p-2 text-center">
        <Stat label={t('accounts.requests')} value={formatNumber(a.requestCount)} />
        <Stat label={t('accounts.tokens')} value={formatNumber(a.totalTokens)} />
        <Stat label={t('accounts.credits')} value={formatNumber(a.totalCredits)} />
        <Stat label={t('stats.errors')} value={formatNumber(a.errorCount)} />
      </div>

      {a.lastUsed > 0 && (
        <p className="text-xs text-muted-foreground">{formatRelativeSeconds(a.lastUsed)}</p>
      )}

      <div className="flex items-center justify-end gap-1">
        <Button variant="ghost" size="sm" onClick={() => onTest(a)} disabled={testing}>
          <FlaskConical className="size-4" />
          {t('accounts.test')}
        </Button>
        <Button variant="ghost" size="icon-sm" onClick={() => onRefresh(a)} disabled={refreshing} aria-label={t('accounts.refresh')}>
          <RefreshCw className={cn('size-4', refreshing && 'animate-spin')} />
        </Button>
        <Button variant="ghost" size="icon-sm" onClick={() => onDetail(a)} aria-label={t('accounts.detail')}>
          <Settings2 className="size-4" />
        </Button>
        <Button variant="ghost" size="icon-sm" onClick={() => onDelete(a)} aria-label={t('accounts.delete')}>
          <Trash2 className="size-4 text-destructive" />
        </Button>
      </div>
    </Card>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-sm font-semibold tabular-nums">{value}</p>
      <p className="text-[11px] text-muted-foreground">{label}</p>
    </div>
  )
}
