import { useState } from 'react'
import { ArrowDown, ArrowUp, GripVertical, ListPlus, Plus, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ModelIcon } from '@/components/shared/ModelBrand'
import { ModelPickerDialog } from './ModelPickerDialog'

interface Props { models: string[]; onChange: (models: string[]) => void; error?: string }

export function OrderedModelPicker({ models, onChange, error }: Props) {
  const { t } = useTranslation()
  const [pickerOpen, setPickerOpen] = useState(false)
  function toggleModel(id: string) {
    const idx = models.findIndex((m) => m.trim().toLowerCase() === id.toLowerCase())
    if (idx >= 0) { onChange(models.filter((_, i) => i !== idx)); return }
    // Fill the first blank free-text slot before appending.
    const blank = models.findIndex((m) => !m.trim())
    if (blank >= 0) onChange(models.map((m, i) => (i === blank ? id : m)))
    else if (models.length < 8) onChange([...models, id])
  }
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
            <span className="flex size-4 shrink-0 items-center justify-center" aria-hidden="true"><ModelIcon model={model} /></span>
            <Input value={model} onChange={(e) => onChange(models.map((m, i) => i === index ? e.target.value : m))} placeholder={t('combos.modelPlaceholder')} aria-label={t('combos.modelAt', { position: index + 1 })} />
            <Button type="button" size="icon" variant="ghost" onClick={() => move(index, -1)} disabled={index === 0} aria-label={t('combos.moveUp')}><ArrowUp className="size-4" /></Button>
            <Button type="button" size="icon" variant="ghost" onClick={() => move(index, 1)} disabled={index === models.length - 1} aria-label={t('combos.moveDown')}><ArrowDown className="size-4" /></Button>
            <Button type="button" size="icon" variant="ghost" onClick={() => onChange(models.filter((_, i) => i !== index))} disabled={models.length === 1} aria-label={t('combos.removeModel')}><X className="size-4" /></Button>
          </div>
        ))}
      </div>
      {models.length < 8 && (
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => setPickerOpen(true)}><ListPlus className="size-4" />{t('combos.pickFromProviders')}</Button>
          <Button type="button" variant="ghost" size="sm" onClick={() => onChange([...models, ''])}><Plus className="size-4" />{t('combos.addModel')}</Button>
        </div>
      )}
      {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
      <p className="text-xs text-muted-foreground">{t('combos.reorderHint')}</p>
      <ModelPickerDialog open={pickerOpen} onOpenChange={setPickerOpen} selected={models} onToggle={toggleModel} max={8} />
    </div>
  )
}
