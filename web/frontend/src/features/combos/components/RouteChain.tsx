import { GitBranch, Scale, Shuffle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ComboStrategy } from '@/types/combo'
import { ModelChip } from '@/components/shared/ModelBrand'

export function RouteChain({ models, strategy, judge }: { models: string[]; strategy: ComboStrategy; judge?: string }) {
  const { t } = useTranslation()
  const Icon = strategy === 'fallback' ? GitBranch : strategy === 'round-robin' ? Shuffle : Scale
  const chain = models.filter(Boolean)
  return (
    <div className="rounded-lg border bg-muted/30 p-3" aria-label={t('combos.routePreview')}>
      <div className="mb-3 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground"><Icon className="size-4" />{t(`combos.strategy.${strategy}`)}</div>
      <div className="flex flex-wrap items-center gap-1.5">
        {chain.map((model, index) => (
          <div key={`${model}-${index}`} className="contents">
            <ModelChip model={model} className="max-w-48 px-2.5 py-1 font-mono text-xs" />
            {index < chain.length - 1 && <span className="text-muted-foreground" aria-hidden="true">{strategy === 'fusion' ? '＋' : '→'}</span>}
          </div>
        ))}
        {strategy === 'fusion' && judge && (
          <>
            <span className="mx-1 text-muted-foreground" aria-hidden="true">⇒</span>
            <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
              {t('combos.judge')}:
              <ModelChip model={judge} className="max-w-48 px-2.5 py-1 font-mono text-xs" />
            </span>
          </>
        )}
      </div>
      <p className="mt-3 text-xs text-muted-foreground">{t(`combos.behavior.${strategy}`)}</p>
    </div>
  )
}
