// SettingsToggleRow — label (+ optional hint) left, control right. min-w-0 so
// long Vietnamese copy wraps instead of colliding with the switch on mobile.
import type { ReactNode } from 'react'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'

interface SettingsToggleRowProps {
  label: string
  hint?: string
  checked?: boolean
  onChange?: (v: boolean) => void
  /** Optional non-switch control (e.g. destructive button). */
  control?: ReactNode
  className?: string
}

export function SettingsToggleRow({
  label,
  hint,
  checked,
  onChange,
  control,
  className,
}: SettingsToggleRowProps) {
  return (
    <div
      className={cn(
        'flex min-w-0 items-start justify-between gap-3 sm:items-center sm:gap-4',
        className,
      )}
    >
      <div className="min-w-0 flex-1">
        <Label className="block whitespace-normal leading-snug">{label}</Label>
        {hint ? (
          <p className="mt-1 text-sm leading-relaxed text-muted-foreground text-pretty">
            {hint}
          </p>
        ) : null}
      </div>
      <div className="shrink-0 pt-0.5 sm:pt-0">
        {control ?? <Switch checked={!!checked} onCheckedChange={onChange} />}
      </div>
    </div>
  )
}
