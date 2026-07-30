import { useEffect, useState } from 'react'
import { ListPlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ApiError } from '@/services/httpClient'
import type { Combo, ComboFieldErrors, ComboInput, ComboStrategy } from '@/types/combo'
import { useCreateCombo, useUpdateCombo } from '@/hooks/mutations/useComboMutations'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { OrderedModelPicker } from './OrderedModelPicker'
import { ModelPickerDialog } from './ModelPickerDialog'
import { RouteChain } from './RouteChain'

const initial: ComboInput = { name: '', strategy: 'fallback', stickyLimit: 1, models: [''], fusionQuorum: 1, fusionTimeoutMs: 30000, judgeModel: '' }
function validate(v: ComboInput): ComboFieldErrors {
  const e: ComboFieldErrors = {}
  if (!/^[A-Za-z0-9_.-]{1,128}$/.test(v.name.trim())) e.name = 'combos.validation.name'
  const clean = v.models.map((m) => m.trim())
  if (clean.some((m) => !m) || new Set(clean.map((m) => m.toLowerCase())).size !== clean.length) e.models = 'combos.validation.models'
  if (v.stickyLimit < 1 || v.stickyLimit > 10000) e.stickyLimit = 'combos.validation.sticky'
  if (v.strategy === 'fusion') {
    if ((v.fusionQuorum ?? 0) < 1 || (v.fusionQuorum ?? 0) > v.models.length) e.fusionQuorum = 'combos.validation.quorum'
    if ((v.fusionTimeoutMs ?? 0) < 100 || (v.fusionTimeoutMs ?? 0) > 300000) e.fusionTimeoutMs = 'combos.validation.timeout'
    if (!v.judgeModel?.trim()) e.judgeModel = 'combos.validation.judge'
  }
  return e
}

export function ComboFormDialog({ open, onOpenChange, editing }: { open: boolean; onOpenChange: (open: boolean) => void; editing: Combo | null }) {
  const { t } = useTranslation()
  const [value, setValue] = useState<ComboInput>(initial)
  const [errors, setErrors] = useState<ComboFieldErrors>({})
  const [judgePickerOpen, setJudgePickerOpen] = useState(false)
  const create = useCreateCombo(); const update = useUpdateCombo()
  useEffect(() => {
    if (!open) return
    setErrors({})
    setValue(editing ? { name: editing.name, strategy: editing.strategy, stickyLimit: editing.stickyLimit, revision: editing.revision, models: editing.models.map((m) => m.model), fusionQuorum: editing.fusionQuorum || 1, fusionTimeoutMs: editing.fusionTimeoutMs || 30000, judgeModel: editing.judgeModel || '' } : initial)
  }, [open, editing])
  function field<K extends keyof ComboInput>(key: K, val: ComboInput[K]) { setValue((v) => ({ ...v, [key]: val })) }
  function submit(event: React.FormEvent) {
    event.preventDefault(); const found = validate(value); setErrors(found); if (Object.keys(found).length) return
    const body: ComboInput = { ...value, name: value.name.trim(), models: value.models.map((m) => m.trim()) }
    if (body.strategy !== 'fusion') { delete body.fusionQuorum; delete body.fusionTimeoutMs; delete body.judgeModel }
    const onSuccess = () => { toast.success(t(editing ? 'combos.updated' : 'combos.created')); onOpenChange(false) }
    const onError = (error: Error) => toast.error(t(error instanceof ApiError && error.status === 409 ? 'combos.conflict' : 'combos.saveFailed'))
    if (editing) update.mutate({ id: editing.id, body }, { onSuccess, onError })
    else create.mutate(body, { onSuccess, onError })
  }
  const pending = create.isPending || update.isPending
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-2xl"><DialogHeader><DialogTitle>{t(editing ? 'combos.edit' : 'combos.create')}</DialogTitle></DialogHeader>
    <form onSubmit={submit} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label htmlFor="combo-name">{t('combos.name')}</Label><Input id="combo-name" autoFocus value={value.name} onChange={(e) => field('name', e.target.value)} aria-invalid={!!errors.name} />{errors.name && <p role="alert" className="text-sm text-destructive">{t(errors.name)}</p>}</div>
      <div className="space-y-2"><Label>{t('combos.strategyLabel')}</Label><Select value={value.strategy} onValueChange={(v) => field('strategy', v as ComboStrategy)}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent>{(['fallback','round-robin','fusion'] as const).map((s) => <SelectItem key={s} value={s}>{t(`combos.strategy.${s}`)}</SelectItem>)}</SelectContent></Select></div></div>
      {value.strategy === 'round-robin' && <div className="space-y-2"><Label htmlFor="sticky">{t('combos.stickyLimit')}</Label><Input id="sticky" type="number" min={1} max={10000} value={value.stickyLimit} onChange={(e) => field('stickyLimit', Number(e.target.value))} />{errors.stickyLimit && <p className="text-sm text-destructive">{t(errors.stickyLimit)}</p>}</div>}
      <OrderedModelPicker models={value.models} onChange={(m) => field('models', m)} error={errors.models ? t(errors.models) : undefined} />
      {value.strategy === 'fusion' && <div className="grid gap-4 rounded-lg border p-4 sm:grid-cols-3"><div className="space-y-2"><Label htmlFor="quorum">{t('combos.quorum')}</Label><Input id="quorum" type="number" min={1} max={value.models.length} value={value.fusionQuorum} onChange={(e) => field('fusionQuorum', Number(e.target.value))} />{errors.fusionQuorum && <p className="text-xs text-destructive">{t(errors.fusionQuorum)}</p>}</div><div className="space-y-2"><Label htmlFor="timeout">{t('combos.timeout')}</Label><Input id="timeout" type="number" min={100} max={300000} value={value.fusionTimeoutMs} onChange={(e) => field('fusionTimeoutMs', Number(e.target.value))} /></div><div className="space-y-2"><Label htmlFor="judge">{t('combos.judgeModel')}</Label><div className="flex gap-2"><Input id="judge" value={value.judgeModel} onChange={(e) => field('judgeModel', e.target.value)} placeholder={t('combos.modelPlaceholder')} /><Button type="button" variant="outline" size="icon" onClick={() => setJudgePickerOpen(true)} aria-label={t('combos.pickFromProviders')}><ListPlus className="size-4" /></Button></div>{errors.judgeModel && <p className="text-xs text-destructive">{t(errors.judgeModel)}</p>}<ModelPickerDialog open={judgePickerOpen} onOpenChange={setJudgePickerOpen} selected={value.judgeModel ? [value.judgeModel] : []} onToggle={(id) => field('judgeModel', id)} single /></div><p className="sm:col-span-3 text-xs text-muted-foreground">{t('combos.fusionWarning')}</p></div>}
      <RouteChain models={value.models} strategy={value.strategy} judge={value.judgeModel} />
      <DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>{t('common.cancel')}</Button><Button type="submit" disabled={pending}>{t('common.save')}</Button></DialogFooter>
    </form></DialogContent></Dialog>
}
