// UsageBar — a labeled progress bar for quota/usage. Shared by account cards and
// API-key rows. Color shifts to amber ≥80% and red ≥95% (matching the old app).
// With a `label`, it renders a label row (label + used/limit) above the bar.
import { clampPercent, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

interface UsageBarProps {
  used: number
  limit: number
  label?: string
  className?: string
  /** Format helper for the used/limit text (defaults to formatNumber). */
  format?: (n: number) => string
}

export function UsageBar({ used, limit, label, className, format = formatNumber }: UsageBarProps) {
  if (!limit || limit <= 0) return null
  const pct = clampPercent(used, limit)
  const fill =
    pct >= 95 ? 'bg-destructive' : pct >= 80 ? 'bg-amber-500' : 'bg-primary'

  return (
    <div className={cn('space-y-1', className)}>
      {label && (
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>{label}</span>
          <span className="tabular-nums">
            {format(used)} / {format(limit)}
          </span>
        </div>
      )}
      <div
        className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(pct)}
      >
        <div className={cn('h-full rounded-full transition-all', fill)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}
