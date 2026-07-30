import { GitBranch, Scale, Shuffle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ComboStrategy } from '@/types/combo'

export function RouteChain({ models, strategy, judge }: { models: string[]; strategy: ComboStrategy; judge?: string }) {
  const { t } = useTranslation()
  const Icon = strategy === 'fallback' ? GitBranch : strategy === 'round-robin' ? Shuffle : Scale
  return (
    <div className="rounded-lg border bg-muted/30 p-3" aria-label={t('combos.routePreview')}>
      <div className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground"><Icon className="size-4" />{t(`combos.strategy.${strategy}`)}</div>
      <div className="flex flex-wrap items-center gap-1.5">
        {models.filter(Boolean).map((model, index) => (
          <div key={`${model}-${index}`} className="contents">
            <span className="max-w-48 truncate rounded-md border bg-background px-2.5 py-1.5 font-mono text-xs" title={model}>{model}</span>
            {index < models.filter(Boolean).length - 1 && <span className="text-muted-foreground" aria-hidden="true">{strategy === 'fusion' ? '＋' : '→'}</span>}
          </div>
        ))}
        {strategy === 'fusion' && judge && <><span className="mx-1 text-muted-foreground" aria-hidden="true">⇒</span><span className="rounded-md border border-primary/30 bg-primary/5 px-2.5 py-1.5 font-mono text-xs">{t('combos.judge')}: {judge}</span></>}
      </div>
      <p className="mt-3 text-xs text-muted-foreground">{t(`combos.behavior.${strategy}`)}</p>
    </div>
  )
}
