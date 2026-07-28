// ProviderModelsPanel — dialog listing a provider bucket's supported models with
// search + copy-id. Opens when `provider` is non-null; fetches the catalog via
// useProviderModels. Model ids are copyable (CopyButton).
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { CopyButton } from '@/components/shared/CopyButton'
import { useProviderModels } from '@/hooks/queries/useProviderModels'
import { providerMeta } from '@/config/providers'

interface Props {
  provider: string | null
  onClose: () => void
}

export function ProviderModelsPanel({ provider, onClose }: Props) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const query = useProviderModels(provider ?? '')

  const meta = provider ? providerMeta(provider) : undefined
  const models = query.data?.models ?? []

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return models
    return models.filter(
      (m) =>
        m.id.toLowerCase().includes(kw) ||
        m.name.toLowerCase().includes(kw) ||
        m.description.toLowerCase().includes(kw),
    )
  }, [models, keyword])

  return (
    <Dialog open={!!provider} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>
            {meta ? t(meta.labelKey) : ''} · {t('providers.supportedModels')}
          </DialogTitle>
        </DialogHeader>

        <div className="relative">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t('providers.searchModels')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>

        <div className="max-h-[60vh] overflow-auto">
          {query.isPending ? (
            <HamsterLoader label={t('detail.loading')} />
          ) : query.isError ? (
            <EmptyState title={t('providers.modelsLoadFailed')} />
          ) : filtered.length === 0 ? (
            <EmptyState title={t('providers.noModels')} />
          ) : (
            <ul className="divide-y">
              {filtered.map((m) => (
                <li key={m.id} className="flex items-center justify-between gap-3 py-2.5">
                  <div className="min-w-0">
                    <p className="truncate font-medium">{m.name || m.id}</p>
                    <p className="truncate font-mono text-xs text-muted-foreground">{m.id}</p>
                  </div>
                  <CopyButton value={m.id} label={t('providers.copyModel')} />
                </li>
              ))}
            </ul>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
