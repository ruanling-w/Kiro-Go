// ModelPickerDialog — pick combo models from the catalogs of logged-in providers,
// grouped by provider (like the reference "Add Model to Combo" flow). Click a model
// to add it, click again to remove. Selection is reflected live via `selected`.
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Search } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { brandFor } from '@/components/shared/ModelBrand'
import { cn } from '@/lib/utils'
import { useComboModelOptions } from '@/hooks/queries/useComboModelOptions'

import type { ComboCandidate } from '@/types/combo'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  selected: ComboCandidate[]
  onToggle: (candidate: ComboCandidate) => void
  /** Cap enforced by the combo form (8). Adding is blocked once reached. */
  max?: number
  /** Single-select mode for the judge field: picking closes the dialog. */
  single?: boolean
}

export function ModelPickerDialog({ open, onOpenChange, selected, onToggle, max, single }: Props) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const { groups, isLoading, hasActiveProviders } = useComboModelOptions()
  const candidateKey = (provider: string | undefined, model: string) => `${provider?.trim().toLowerCase() ?? ''}\0${model.trim().toLowerCase()}`
  const selectedSet = useMemo(() => new Set(selected.map((m) => candidateKey(m.provider, m.model))), [selected])

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase()
    if (!kw) return groups
    return groups
      .map((g) => ({
        ...g,
        models: g.models.filter(
          (m) =>
            m.id.toLowerCase().includes(kw) ||
            m.name.toLowerCase().includes(kw) ||
            m.description.toLowerCase().includes(kw),
        ),
      }))
      .filter((g) => g.models.length > 0)
  }, [groups, keyword])

  const atCap = typeof max === 'number' && selected.length >= max

  function pick(provider: string, model: string) {
    const active = selectedSet.has(candidateKey(provider, model))
    if (!active && !single && atCap) return
    onToggle({ provider, model })
    if (single) onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[calc(100%-2rem)] sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('combos.pickerTitle')}</DialogTitle>
        </DialogHeader>

        <p className="rounded-md bg-muted/60 px-3 py-2 text-xs text-muted-foreground">
          {single ? t('combos.pickerHintSingle') : t('combos.pickerHint')}
        </p>

        <div className="relative">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t('providers.searchModels')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
        </div>

        <div className="max-h-[55vh] overflow-auto">
          {isLoading ? (
            <HamsterLoader label={t('detail.loading')} />
          ) : !hasActiveProviders ? (
            <EmptyState title={t('combos.pickerNoProviders')} />
          ) : filtered.length === 0 ? (
            <EmptyState title={t('providers.noModels')} />
          ) : (
            <div className="space-y-4">
              {filtered.map((group) => (
                <div key={group.key} className="space-y-1.5">
                  <div className="flex items-center gap-2 px-1 text-sm font-medium">
                    <ProviderIcon provider={group.key} className="size-4" />
                    <span>{t(group.labelKey)}</span>
                    <span className="text-xs text-muted-foreground">({group.models.length})</span>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {group.models.map((m) => {
                      const active = selectedSet.has(candidateKey(group.key, m.id))
                      const disabled = !active && !single && atCap
                      // Unselected chips wear their provider palette (same one the
                      // log table uses); selection overrides it with the primary tint.
                      const brand = brandFor(group.key, m.id, t)
                      return (
                        <button
                          key={`${group.key}:${m.id}`}
                          type="button"
                          onClick={() => pick(group.key, m.id)}
                          disabled={disabled}
                          title={m.description || m.id}
                          aria-pressed={active}
                          className={cn(
                            'inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm transition-colors',
                            active
                              ? 'border-primary bg-primary/10 text-primary'
                              : brand
                                ? cn(brand.chip, brand.text, 'hover:brightness-110')
                                : 'border-border hover:bg-muted',
                            disabled && 'cursor-not-allowed opacity-50',
                          )}
                        >
                          {active ? (
                            <Check className="size-3.5 shrink-0" />
                          ) : (
                            <ProviderIcon provider={group.key} className="size-3.5 shrink-0" />
                          )}
                          <span className="max-w-[16rem] truncate">{m.name || m.id}</span>
                        </button>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
