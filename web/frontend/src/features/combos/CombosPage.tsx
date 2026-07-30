import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, GitFork, MoreHorizontal, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import type { Combo } from '@/types/combo'
import { ApiError } from '@/services/httpClient'
import { useCombos } from '@/hooks/queries/useCombos'
import { useDeleteCombo, useResetComboRotation } from '@/hooks/mutations/useComboMutations'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { ComboFormDialog } from './components/ComboFormDialog'
import { RouteChain } from './components/RouteChain'

export default function CombosPage() {
  const { t } = useTranslation(); const query = useCombos(); const remove = useDeleteCombo(); const reset = useResetComboRotation(); const clipboard = useCopyToClipboard()
  const [formOpen, setFormOpen] = useState(false); const [editing, setEditing] = useState<Combo | null>(null); const [deleting, setDeleting] = useState<Combo | null>(null); const [resetting, setResetting] = useState<Combo | null>(null)
  function openEdit(combo: Combo) { setEditing(combo); setFormOpen(true) }
  function mutationError(error: Error) { toast.error(t(error instanceof ApiError && error.status === 409 ? 'combos.conflict' : 'common.failed')) }
  return <div className="space-y-5"><PageHeader title={t('combos.title')} description={t('combos.subtitle')} actions={<Button onClick={() => { setEditing(null); setFormOpen(true) }}><Plus className="size-4" />{t('combos.create')}</Button>} />
    {query.isPending ? <HamsterLoader label={t('detail.loading')} /> : query.isError ? <Card><CardContent className="py-10 text-center"><p className="text-sm text-destructive">{t('combos.loadFailed')}</p><Button className="mt-4" variant="outline" onClick={() => void query.refetch()}>{t('common.retry')}</Button></CardContent></Card> : !query.data?.length ? <EmptyState icon={GitFork} title={t('combos.emptyTitle')} description={t('combos.emptyDescription')} action={<Button onClick={() => setFormOpen(true)}><Plus className="size-4" />{t('combos.create')}</Button>} /> :
    <div className="grid gap-4 xl:grid-cols-2">{query.data.map((combo) => <Card key={combo.id} className="overflow-hidden"><CardHeader className="flex flex-row items-start justify-between gap-3"><div className="min-w-0"><CardTitle className="flex flex-wrap items-center gap-2"><span className="truncate">{combo.name}</span><Badge variant="secondary">{t(`combos.strategy.${combo.strategy}`)}</Badge></CardTitle><button className="mt-1 flex max-w-full items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => { void clipboard.copy(combo.id); toast.success(t('combos.copied')) }} title={combo.id}><span className="truncate">{combo.id}</span><Copy className="size-3 shrink-0" /></button></div><DropdownMenu><DropdownMenuTrigger asChild><Button size="icon" variant="ghost" aria-label={t('common.actions')}><MoreHorizontal className="size-4" /></Button></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuItem onClick={() => openEdit(combo)}><Pencil className="size-4" />{t('combos.edit')}</DropdownMenuItem>{combo.strategy === 'round-robin' && <DropdownMenuItem onClick={() => setResetting(combo)}><RefreshCw className="size-4" />{t('combos.reset')}</DropdownMenuItem>}<DropdownMenuItem className="text-destructive" onClick={() => setDeleting(combo)}><Trash2 className="size-4" />{t('common.delete')}</DropdownMenuItem></DropdownMenuContent></DropdownMenu></CardHeader><CardContent><RouteChain models={combo.models.map((m) => m.model)} strategy={combo.strategy} judge={combo.judgeModel} /><p className="mt-3 text-xs text-muted-foreground">{t('combos.revision', { revision: combo.revision })}</p></CardContent></Card>)}</div>}
    <ComboFormDialog open={formOpen} onOpenChange={setFormOpen} editing={editing} />
    <ConfirmDialog open={!!deleting} onOpenChange={(o) => !o && setDeleting(null)} title={t('combos.deleteTitle')} description={t('combos.deleteDescription', { name: deleting?.name })} confirmLabel={t('common.delete')} destructive onConfirm={() => { if (!deleting) return; remove.mutate({ id: deleting.id, revision: deleting.revision }, { onSuccess: () => toast.success(t('combos.deleted')), onError: mutationError }); setDeleting(null) }} />
    <ConfirmDialog open={!!resetting} onOpenChange={(o) => !o && setResetting(null)} title={t('combos.resetTitle')} description={t('combos.resetDescription', { name: resetting?.name })} confirmLabel={t('combos.reset')} onConfirm={() => { if (!resetting) return; reset.mutate({ id: resetting.id, revision: resetting.revision }, { onSuccess: () => toast.success(t('combos.resetDone')), onError: mutationError }); setResetting(null) }} />
  </div>
}
