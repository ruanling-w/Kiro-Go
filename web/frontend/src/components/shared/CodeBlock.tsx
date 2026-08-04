// CodeBlock — mono pre with optional filename header and copy button.
// No syntax highlighter; Tailwind tokens only so light/dark both work.
import { CopyButton } from '@/components/shared/CopyButton'
import { cn } from '@/lib/utils'

interface CodeBlockProps {
  code: string
  lang?: string
  filename?: string
  className?: string
  /** Accessible label for the copy button. */
  copyLabel?: string
}

export function CodeBlock({
  code,
  lang,
  filename,
  className,
  copyLabel,
}: CodeBlockProps) {
  return (
    <div
      className={cn(
        'min-w-0 overflow-hidden rounded-lg border border-border/60 bg-muted/40',
        className,
      )}
    >
      {(filename || lang) && (
        <div className="flex items-center justify-between gap-2 border-b border-border/60 bg-muted/60 px-3 py-1.5">
          <div className="min-w-0 truncate font-mono text-xs text-muted-foreground">
            {filename ?? lang}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {filename && lang ? (
              <span className="hidden text-[10px] uppercase tracking-wide text-muted-foreground/80 sm:inline">
                {lang}
              </span>
            ) : null}
            <CopyButton value={code} size="icon-xs" label={copyLabel} />
          </div>
        </div>
      )}
      {!filename && !lang ? (
        <div className="flex justify-end px-2 pt-2">
          <CopyButton value={code} size="icon-xs" label={copyLabel} />
        </div>
      ) : null}
      {/* Own horizontal scroll so long curl/json lines don't blow out the page. */}
      <pre className="max-h-[22rem] overflow-x-auto overflow-y-auto overscroll-x-contain p-3 text-[11px] leading-relaxed sm:max-h-[28rem] sm:text-[13px]">
        <code className="font-mono whitespace-pre text-foreground/90">{code}</code>
      </pre>
    </div>
  )
}
