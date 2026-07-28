// StatCard — a KPI tile: label, value, optional icon/hint/trend and children
// (e.g. a sparkline). Used across Dashboard and page headers.
import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface StatCardProps {
  label: string
  value: ReactNode
  icon?: LucideIcon
  hint?: string
  className?: string
  children?: ReactNode
}

export function StatCard({ label, value, icon: Icon, hint, className, children }: StatCardProps) {
  return (
    <Card className={cn('flex flex-col gap-1 p-4', className)}>
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
        {Icon && <Icon className="size-4 text-muted-foreground" />}
      </div>
      <div className="text-2xl font-semibold tabular-nums">{value}</div>
      {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      {children}
    </Card>
  )
}
