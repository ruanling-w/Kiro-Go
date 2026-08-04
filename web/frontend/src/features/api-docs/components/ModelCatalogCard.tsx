// ModelCatalogCard — searchable list of ids from public /v1/models.
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { CopyButton } from '@/components/shared/CopyButton'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { useThinking } from '@/hooks/queries/useSettings'
import type { PublicModel } from '@/types/publicApi'
import { tp } from '@/lib/t'

interface ModelCatalogCardProps {
  models: PublicModel[]
  loading: boolean
  error: boolean
}

export function ModelCatalogCard({ models, loading, error }: ModelCatalogCardProps) {
  const { t } = useTranslation()
  const thinking = useThinking()
  const suffix = thinking.data?.suffix || '-thinking'
  const [q, setQ] = useState('')

  const filtered = useMemo(() => {
    const kw = q.trim().toLowerCase()
    if (!kw) return models
    return models.filter((m) => m.id.toLowerCase().includes(kw))
  }, [models, q])

  return (
    <Card className="min-w-0">
      <CardHeader className="gap-1.5">
        <CardTitle>{t('api.modelList')}</CardTitle>
        <CardDescription>{t('apiDocs.modelsDesc')}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0 space-y-3">
        <p className="text-xs leading-relaxed text-muted-foreground">
          {tp(t, 'apiDocs.thinkingHint', suffix)}
        </p>

        <div className="relative min-w-0">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t('api.searchModels')}
            className="min-w-0 pl-8"
          />
        </div>

        {loading ? (
          <HamsterLoader size="sm" label={t('api.loading')} />
        ) : error ? (
          <EmptyState title={t('api.fetchError')} />
        ) : filtered.length === 0 ? (
          <EmptyState title={t('api.noModels')} />
        ) : (
          <ul className="max-h-[22rem] divide-y overflow-auto rounded-lg border border-border/60 md:max-h-[36rem]">
            {filtered.map((m) => (
              <li key={m.id} className="flex items-start gap-2 px-3 py-2.5 sm:items-center">
                <div className="min-w-0 flex-1">
                  <p className="break-all font-mono text-sm leading-snug sm:truncate sm:break-normal">
                    {m.id}
                  </p>
                  <div className="mt-1 flex flex-wrap items-center gap-1.5">
                    {m.owned_by ? (
                      <span className="text-[11px] text-muted-foreground">{m.owned_by}</span>
                    ) : null}
                    {m.combo ? <StatusBadge tone="info">combo</StatusBadge> : null}
                    {m.strategy ? (
                      <StatusBadge tone="neutral">{m.strategy}</StatusBadge>
                    ) : null}
                    {m.supports_image || m.capabilities?.vision ? (
                      <StatusBadge tone="success">vision</StatusBadge>
                    ) : null}
                  </div>
                </div>
                <CopyButton
                  value={m.id}
                  size="icon-xs"
                  label={t('common.copy')}
                  className="mt-0.5 shrink-0 sm:mt-0"
                />
              </li>
            ))}
          </ul>
        )}

        {!loading && !error && models.length > 0 ? (
          <p className="text-xs text-muted-foreground">
            {t('api.totalModels', { count: filtered.length })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
