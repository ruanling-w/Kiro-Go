// StatusBadge — colored pill for account/api-key/log status. Tone maps to a
// semantic color; label is caller-provided (already translated).
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

export type StatusTone = 'success' | 'warning' | 'danger' | 'neutral' | 'info'

const TONE: Record<StatusTone, string> = {
  success:
    'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  warning: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400',
  danger: 'border-destructive/30 bg-destructive/10 text-destructive',
  info: 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-400',
  neutral: 'border-border bg-muted text-muted-foreground',
}

interface StatusBadgeProps {
  tone: StatusTone
  children: React.ReactNode
  className?: string
}

export function StatusBadge({ tone, children, className }: StatusBadgeProps) {
  return (
    <Badge variant="outline" className={cn('font-medium', TONE[tone], className)}>
      {children}
    </Badge>
  )
}
