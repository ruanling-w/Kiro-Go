import { ArrowDown, ArrowUp, GripVertical, Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface Props { models: string[]; onChange: (models: string[]) => void; error?: string }

export function OrderedModelPicker({ models, onChange, error }: Props) {
  const { t } = useTranslation()
  function move(index: number, delta: number) {
    const nextIndex = index + delta
    if (nextIndex < 0 || nextIndex >= models.length) return
    const next = [...models]
    ;[next[index], next[nextIndex]] = [next[nextIndex], next[index]]
    onChange(next)
  }
  function handleKeyDown(event: React.KeyboardEvent, index: number) {
    if (event.altKey && (event.key === 'ArrowUp' || event.key === 'ArrowDown')) {
      event.preventDefault()
      move(index, event.key === 'ArrowUp' ? -1 : 1)
    }
  }
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t('combos.models')}</span>
        <span className="text-xs text-muted-foreground">{t('combos.modelCount', { count: models.length })}</span>
      </div>
      <div className="space-y-2" role="list" aria-label={t('combos.models')}>
        {models.map((model, index) => (
          <div key={index} role="listitem" className="flex items-center gap-2" onKeyDown={(e) => handleKeyDown(e, index)}>
            <span className="flex size-8 shrink-0 items-center justify-center rounded-md border bg-muted font-mono text-xs" aria-hidden="true">{index + 1}</span>
            <GripVertical className="hidden size-4 text-muted-foreground sm:block" aria-hidden="true" />
            <Input value={model} onChange={(e) => onChange(models.map((m, i) => i === index ? e.target.value : m))} placeholder={t('combos.modelPlaceholder')} aria-label={t('combos.modelAt', { position: index + 1 })} />
            <Button type="button" size="icon" variant="ghost" onClick={() => move(index, -1)} disabled={index === 0} aria-label={t('combos.moveUp')}><ArrowUp className="size-4" /></Button>
            <Button type="button" size="icon" variant="ghost" onClick={() => move(index, 1)} disabled={index === models.length - 1} aria-label={t('combos.moveDown')}><ArrowDown className="size-4" /></Button>
            <Button type="button" size="icon" variant="ghost" onClick={() => onChange(models.filter((_, i) => i !== index))} disabled={models.length === 1} aria-label={t('combos.removeModel')}><X className="size-4" /></Button>
          </div>
        ))}
      </div>
      {models.length < 8 && <Button type="button" variant="outline" size="sm" onClick={() => onChange([...models, ''])}><Plus className="size-4" />{t('combos.addModel')}</Button>}
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      <p className="text-xs text-muted-foreground">{t('combos.reorderHint')}</p>
    </div>
  )
}
