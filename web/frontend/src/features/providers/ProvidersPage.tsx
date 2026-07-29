// Providers landing — a grid of provider buckets (kiro/antigravity/grok/codex/
// remotekiro) plus an "All accounts" card, each with a live account count.
// Clicking a bucket routes to its own page (/providers/:key); the "All accounts"
// card routes to the shared /accounts list. "Models" opens the ProviderModelsPanel.
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Layers } from 'lucide-react'
import { useAccounts } from '@/hooks/queries/useAccounts'
import { PROVIDERS, ALL_PROVIDER, bucketOf } from '@/config/providers'
import { PageHeader } from '@/components/shared/PageHeader'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { cn } from '@/lib/utils'
import { ProviderModelsPanel } from './components/ProviderModelsPanel'

export default function ProvidersPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const accounts = useAccounts()
  const [modelsFor, setModelsFor] = useState<string | null>(null)

  const list = accounts.data ?? []
  const counts = new Map<string, number>()
  for (const a of list) {
    const b = bucketOf(a.provider)
    counts.set(b, (counts.get(b) ?? 0) + 1)
  }

  return (
    <div className="space-y-6">
      <PageHeader title={t('nav.providers')} />

      {accounts.isPending ? (
        <HamsterLoader label={t('detail.loading')} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Card
            role="button"
            tabIndex={0}
            onClick={() => navigate('/accounts')}
            onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && navigate('/accounts')}
            className="flex cursor-pointer flex-col justify-between gap-4 p-5 transition-colors hover:bg-muted/50"
          >
            <div className="flex items-center gap-3">
              <div className={cn('flex size-11 items-center justify-center rounded-xl bg-muted', ALL_PROVIDER.color)}>
                <Layers className="size-6" />
              </div>
              <div>
                <h3 className="font-semibold">{t('nav.allAccounts')}</h3>
                <p className="text-sm text-muted-foreground">{t('providers.allDesc')}</p>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-2xl font-semibold tabular-nums">{list.length}</span>
            </div>
          </Card>

          {PROVIDERS.map((p) => {
            const count = counts.get(p.key) ?? 0
            return (
              <Card
                key={p.key}
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/providers/${p.key}`)}
                onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && navigate(`/providers/${p.key}`)}
                className="flex cursor-pointer flex-col justify-between gap-4 p-5 transition-colors hover:bg-muted/50"
              >
                <div className="flex items-center gap-3">
                  <div className={cn('flex size-11 items-center justify-center rounded-xl bg-muted', p.color)}>
                    <ProviderIcon provider={p.key} className="size-6" />
                  </div>
                  <div>
                    <h3 className="font-semibold">{t(p.labelKey)}</h3>
                    <p className="text-sm text-muted-foreground">{t(p.descKey)}</p>
                  </div>
                </div>
                <div className="flex items-center justify-between gap-2">
                  <span className="text-2xl font-semibold tabular-nums">{count}</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      setModelsFor(p.key)
                    }}
                  >
                    {t('providers.supportedModels')}
                  </Button>
                </div>
              </Card>
            )
          })}
        </div>
      )}

      <ProviderModelsPanel provider={modelsFor} onClose={() => setModelsFor(null)} />
    </div>
  )
}
