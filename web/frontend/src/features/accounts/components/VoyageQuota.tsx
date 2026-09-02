import type { VoyageQuotaBucket } from '@/types/account'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

interface Props {
  buckets?: VoyageQuotaBucket[]
  detail?: boolean
}

export function VoyageQuota({ buckets, detail = false }: Props) {
  const items = Array.isArray(buckets) ? buckets : []

  // Filter buckets: on card show used models + top popular models; on detail show all
  const activeItems = detail
    ? items
    : items.filter((b) => b.usedTokens > 0 || ['voyage-4-large', 'rerank-2.5', 'voyage-finance-2'].includes(b.model))

  if (activeItems.length === 0) {
    return (
      <div className="rounded-lg border border-dashed px-3 py-2 text-xs text-muted-foreground">
        Hạn mức Free Tier: 200M tokens (Embeddings & Reranker), 50M tokens (Finance / Law).
      </div>
    )
  }

  return (
    <section className="space-y-2 rounded-lg border bg-muted/20 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="font-semibold text-foreground">Hạn mức Free Tier từng Model</span>
        <span className="text-muted-foreground text-[11px]">
          {items.some((b) => b.usedTokens > 0)
            ? 'Đang theo dõi tự động'
            : '200M / 50M Free Tokens'}
        </span>
      </div>

      <div className={cn('grid gap-2', detail && 'sm:grid-cols-2')}>
        {activeItems.map((b) => {
          const usedTokens = b.usedTokens || 0
          const freeLimit = b.freeLimitTokens || 0
          const usedPct = Math.min(100, Math.max(0, b.usedPercent || 0))
          const isFree = freeLimit > 0
          const isExhausted = b.isFreeExhausted

          return (
            <div
              key={b.model}
              className="space-y-1 rounded-md border border-border/60 bg-background/60 p-2 text-xs"
            >
              <div className="flex items-center justify-between gap-1">
                <span className="font-medium text-foreground truncate" title={b.model}>
                  {b.displayName || b.model}
                </span>
                <span
                  className={cn(
                    'text-[10px] font-mono shrink-0',
                    isExhausted ? 'text-destructive font-semibold' : 'text-muted-foreground'
                  )}
                >
                  {isFree
                    ? `${formatNumber(usedTokens)} / ${freeLimit / 1_000_000}M`
                    : `${formatNumber(usedTokens)} tokens (Trả phí)`}
                </span>
              </div>

              {isFree && (
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className={cn(
                      'h-full rounded-full transition-all duration-300',
                      isExhausted
                        ? 'bg-destructive'
                        : usedPct >= 80
                        ? 'bg-amber-500'
                        : 'bg-primary'
                    )}
                    style={{ width: `${Math.max(usedPct, usedTokens > 0 ? 2 : 0)}%` }}
                  />
                </div>
              )}

              <div className="flex items-center justify-between text-[10px] text-muted-foreground">
                <span>
                  {isFree
                    ? `Còn lại: ${formatNumber(b.remainingFreeTokens || freeLimit)}`
                    : `Giá: $${b.pricePerMillion}/M`}
                </span>
                {b.costUsd > 0 && (
                  <span className="font-medium text-amber-500">Phát sinh: ${b.costUsd.toFixed(4)}</span>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}
