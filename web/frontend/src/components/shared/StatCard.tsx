// StatCard — a KPI tile: label, value, optional icon/hint/tone and children
// (e.g. a sparkline). Used across Dashboard and page headers.
//
// Two value modes (non-breaking): pass a ready `value` (ReactNode) for static
// content, OR pass `count` (a number) + optional `format` to get a mono count-up
// readout via <Counter>. `tone` maps to a semantic accent (border + underglow)
// so a KPI can signal health (e.g. success rate → success/warning/danger).
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Card } from '@/components/ui/card'
import { Counter } from '@/components/ui/animate/Counter'
import { cn } from '@/lib/utils'

export type StatTone = 'default' | 'success' | 'warning' | 'danger'

interface StatCardProps {
  label: string
  value?: ReactNode
  count?: number
  format?: (n: number) => string
  icon?: LucideIcon
  hint?: string
  tone?: StatTone
  /** Green pulse “live” dot next to the label (e.g. realtime RPM). */
  live?: boolean
  className?: string
  children?: ReactNode
}

const TONE_RING: Record<StatTone, string> = {
  default: 'ring-foreground/10 hover:ring-primary/30',
  success: 'ring-emerald-500/25 hover:ring-emerald-500/40',
  warning: 'ring-amber-500/25 hover:ring-amber-500/40',
  danger: 'ring-destructive/25 hover:ring-destructive/40',
}

const TONE_GLOW: Record<StatTone, string> = {
  default: 'from-primary/50',
  success: 'from-emerald-500/50',
  warning: 'from-amber-500/50',
  danger: 'from-destructive/50',
}

const TONE_ICON: Record<StatTone, string> = {
  default: 'bg-primary/10 text-primary',
  success: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  warning: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
  danger: 'bg-destructive/10 text-destructive',
}

export function StatCard({
  label,
  value,
  count,
  format,
  icon: Icon,
  hint,
  tone = 'default',
  live = false,
  className,
  children,
}: StatCardProps) {
  const { t } = useTranslation()
  return (
    <Card
      className={cn(
        'group/stat relative flex h-full flex-col gap-1.5 p-4 transition-all duration-200 hover:-translate-y-0.5',
        TONE_RING[tone],
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="min-w-0 truncate text-xs font-medium text-muted-foreground">
          {label}
        </span>
        {Icon && (
          <span
            className={cn(
              'flex size-7 shrink-0 items-center justify-center rounded-lg',
              TONE_ICON[tone],
            )}
          >
            <Icon className="size-4" />
          </span>
        )}
      </div>
      <div className="font-telemetry text-[1.75rem] leading-tight font-semibold">
        {count !== undefined ? <Counter value={count} format={format} /> : value}
      </div>

      {/* Footer block sticks to the bottom so equal-height cards align. */}
      <div className="mt-auto flex min-h-0 flex-col gap-1">
        {(hint || live) && (
          <div className="flex items-end justify-between gap-2">
            {hint ? (
              <span className="min-w-0 flex-1 text-xs leading-snug text-muted-foreground">
                {hint}
              </span>
            ) : (
              <span />
            )}
            {live && (
              <span className="inline-flex shrink-0 items-center gap-1.5 text-[11px] font-medium leading-snug text-emerald-600 dark:text-emerald-400">
                <span className="relative flex size-1.5 shrink-0" aria-hidden>
                  <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-60" />
                  <span className="relative inline-flex size-1.5 rounded-full bg-emerald-500" />
                </span>
                {t('status.live')}
              </span>
            )}
          </div>
        )}
        {children}
      </div>
      <span
        className={cn(
          'pointer-events-none absolute inset-x-0 bottom-0 h-px bg-gradient-to-r to-transparent opacity-0 transition-opacity duration-300 group-hover/stat:opacity-100',
          TONE_GLOW[tone],
        )}
      />
    </Card>
  )
}
