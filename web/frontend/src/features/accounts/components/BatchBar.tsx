// BatchBar — appears when ≥1 account is selected. Enable/disable/refresh run a
// batch mutation; delete goes through the shared ConfirmDialog. Select-all is
// driven by the caller (it knows the currently-filtered id set).
import { useTranslation } from 'react-i18next'
import { CheckCheck, Power, PowerOff, RefreshCw, Trash2, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { tp } from '@/lib/t'

interface Props {
  selectedCount: number
  allSelected: boolean
  onToggleAll: () => void
  onEnable: () => void
  onDisable: () => void
  onRefresh: () => void
  onDelete: () => void
  onClear: () => void
  busy?: boolean
}

export function BatchBar({
  selectedCount,
  allSelected,
  onToggleAll,
  onEnable,
  onDisable,
  onRefresh,
  onDelete,
  onClear,
  busy,
}: Props) {
  const { t } = useTranslation()
  if (selectedCount === 0) return null

  return (
    <div className="sticky top-0 z-10 flex flex-wrap items-center gap-2 rounded-xl border bg-card/95 p-3 shadow-sm backdrop-blur">
      <span className="mr-2 text-sm font-medium">
        {tp(t, 'export.selected', selectedCount)}
      </span>
      <Button variant="outline" size="sm" onClick={onToggleAll}>
        <CheckCheck className="size-4" />
        {allSelected ? t('export.deselectAll') : t('export.selectAll')}
      </Button>
      <div className="mx-1 h-5 w-px bg-border" />
      <Button variant="outline" size="sm" onClick={onEnable} disabled={busy}>
        <Power className="size-4" />
        {t('accounts.enable')}
      </Button>
      <Button variant="outline" size="sm" onClick={onDisable} disabled={busy}>
        <PowerOff className="size-4" />
        {t('accounts.disable')}
      </Button>
      <Button variant="outline" size="sm" onClick={onRefresh} disabled={busy}>
        <RefreshCw className={busy ? 'size-4 animate-spin' : 'size-4'} />
        {t('accounts.refresh')}
      </Button>
      <Button variant="destructive" size="sm" onClick={onDelete} disabled={busy}>
        <Trash2 className="size-4" />
        {t('accounts.delete')}
      </Button>
      <Button variant="ghost" size="icon-sm" className="ml-auto" onClick={onClear} aria-label={t('common.close')}>
        <X className="size-4" />
      </Button>
    </div>
  )
}
