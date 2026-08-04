// HelpBlock — the "how do I get this token?" guidance the legacy modals rendered
// inline (web/js/auth-modals.js). The React flows dropped it even though every
// locale string survived, so imports shipped as bare inputs with no instructions.
// Collapsible so it explains without crowding out the form.
import { useState, type ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface Props {
  title: string
  children: ReactNode
  /** Start expanded — use for flows where the steps are the whole point. */
  defaultOpen?: boolean
}

export function HelpBlock({ title, children, defaultOpen = true }: Props) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="rounded-lg border bg-muted/40 text-sm">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left font-medium"
        aria-expanded={open}
      >
        <span>{title}</span>
        <ChevronDown className={cn('size-4 shrink-0 transition-transform', open && 'rotate-180')} />
      </button>
      {open && <div className="space-y-1.5 px-3 pb-3 text-muted-foreground">{children}</div>}
    </div>
  )
}

/** Numbered step list, matching the legacy `.steps-list` markup. */
export function Steps({ children }: { children: ReactNode }) {
  return <ol className="list-decimal space-y-1 pl-5">{children}</ol>
}

/** Inline monospace snippet for file paths / cookie names. */
export function Code({ children }: { children: ReactNode }) {
  return (
    <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs break-all">{children}</code>
  )
}
