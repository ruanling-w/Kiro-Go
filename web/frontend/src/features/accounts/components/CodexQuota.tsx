import { useTranslation } from 'react-i18next'
import type { CodexQuotaWindow } from '@/types/account'
import { tp } from '@/lib/t'
import { cn } from '@/lib/utils'

interface Props {
  windows?: CodexQuotaWindow[]
  limitReached?: boolean
  resetCredits?: number
  detail?: boolean
}

function percent(value: unknown): number {
  const n = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(n)) return 0
  return Math.min(100, Math.max(0, n))
}

function resetLabel(value: string | undefined): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function CodexQuota({ windows, limitReached, resetCredits, detail = false }: Props) {
  const { t } = useTranslation()
  const items = Array.isArray(windows) ? windows : []
  const credits = Number.isFinite(resetCredits) ? Number(resetCredits) : 0

  if (items.length === 0 && !limitReached && credits <= 0) {
    return (
      <div className="rounded-lg border border-dashed px-3 py-2 text-xs text-muted-foreground">
        {t('codex.quotaEmpty')}
      </div>
    )
  }

  return (
    <section className="space-y-2 rounded-lg border bg-muted/20 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="font-semibold text-foreground">{t('codex.quotaTitle')}</span>
        <span className={cn('text-muted-foreground', limitReached && 'font-medium text-destructive')}>
          {limitReached
            ? t('codex.limitReached')
            : tp(t, 'codex.quotaSummary', items.length)}
        </span>
      </div>

      <div className={cn('grid gap-2', detail && items.length > 1 && 'sm:grid-cols-2')}>
        {items.map((window, index) => {
          const used = percent(window.usedPct)
          const reset = resetLabel(window.resetAt)
          const hit = Boolean(window.limitHit)
          const translatedKey = `codex.window.${window.key}`
          const translatedLabel = t(translatedKey)
          const label = translatedLabel === translatedKey
            ? window.label || window.key || '-'
            : translatedLabel
          const fill = hit || used >= 95
            ? 'bg-destructive'
            : used >= 80
              ? 'bg-amber-500'
              : 'bg-primary'

          return (
            <div key={`${window.key || 'quota'}-${index}`} className="space-y-1">
              <div className="flex min-w-0 items-center justify-between gap-2 text-xs">
                <span className={cn('truncate text-muted-foreground', hit && 'font-medium text-destructive')}>
                  {label}
                </span>
                <span className={cn('shrink-0 tabular-nums', hit && 'font-semibold text-destructive')}>
                  {Math.round(used)}%
                </span>
              </div>
              <div
                className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
                role="progressbar"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={Math.round(used)}
                aria-label={label}
              >
                <div className={cn('h-full rounded-full transition-all', fill)} style={{ width: `${used}%` }} />
              </div>
              {(reset || hit) && (
                <p className={cn('text-[11px] text-muted-foreground', hit && 'text-destructive')}>
                  {[reset ? tp(t, 'codex.quotaReset', reset) : '', hit ? t('codex.limitReached') : '']
                    .filter(Boolean)
                    .join(' · ')}
                </p>
              )}
            </div>
          )
        })}
      </div>

      {credits > 0 && (
        <p className="text-xs text-muted-foreground">
          {detail ? `${t('codex.resetCreditsLabel')}: ${credits}` : tp(t, 'codex.resetCredits', credits)}
        </p>
      )}
    </section>
  )
}
