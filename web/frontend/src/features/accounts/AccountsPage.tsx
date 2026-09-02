import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, Eye, EyeOff, RotateCw, Download } from 'lucide-react'
import type { AccountListItem, ProviderKey } from '@/types/account'
import { useAccounts } from '@/hooks/queries/useAccounts'
import {
  useDeleteAccount,
  useRefreshAccount,
  useBatchAccounts,
  useConsumeCodexResetCredit,
} from '@/hooks/mutations/useAccountMutations'
import { PageHeader } from '@/components/shared/PageHeader'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { useDebounce } from '@/hooks/useDebounce'
import { useUiStore } from '@/stores/uiStore'
import { bucketOf, providerMeta } from '@/config/providers'
import { tp } from '@/lib/t'
import { toast } from 'sonner'
import { AccountFilters } from './components/AccountFilters'
import { BatchBar } from './components/BatchBar'
import { AccountCard } from './components/AccountCard'
import { AccountDetailDialog } from './components/AccountDetailDialog'
import { TestAccountDialog } from './components/TestAccountDialog'
import { ExportDialog } from './components/ExportDialog'
import { AddAccountDialog } from '@/features/auth-modals/AddAccountDialog'

const PAGE_SIZE = 24

function isBanned(a: AccountListItem): boolean {
  return !!a.banStatus && a.banStatus !== 'none' && a.banStatus !== ''
}

export default function AccountsPage() {
  const { t } = useTranslation()
  const { provider: routeParam } = useParams()
  const accounts = useAccounts()
  const deleteAccount = useDeleteAccount()
  const refreshAccount = useRefreshAccount()

  const batch = useBatchAccounts()

  // On a provider-specific page (/providers/:provider) the bucket is locked from
  // the URL; the shared /accounts page falls back to the uiStore filter dropdown.
  const lockedMeta = routeParam ? providerMeta(routeParam) : undefined
  const lockedProvider = lockedMeta?.key
  const keyword = useUiStore((s) => s.accountKeyword)
  const status = useUiStore((s) => s.accountStatus)
  const storeProvider = useUiStore((s) => s.providerFilter)
  const provider: ProviderKey | '' = lockedProvider ?? (storeProvider as ProviderKey | '')
  const privacy = useUiStore((s) => s.privacyMode)
  const togglePrivacy = useUiStore((s) => s.togglePrivacy)
  const selected = useUiStore((s) => s.selectedAccounts)
  const setAccountSelection = useUiStore((s) => s.setAccountSelection)
  const clearAccountSelection = useUiStore((s) => s.clearAccountSelection)
  const debouncedKeyword = useDebounce(keyword, 300)

  const [page, setPage] = useState(0)
  const [pendingDelete, setPendingDelete] = useState<AccountListItem | null>(null)
  const [pendingResetQuota, setPendingResetQuota] = useState<AccountListItem | null>(null)
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false)
  const [testTarget, setTestTarget] = useState<AccountListItem | null>(null)
  const [detailTarget, setDetailTarget] = useState<AccountListItem | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [exportOpen, setExportOpen] = useState(false)

  const consumeResetCredit = useConsumeCodexResetCredit()

  const list = accounts.data ?? []

  const filtered = useMemo(() => {
    const kw = debouncedKeyword.trim().toLowerCase()
    return list.filter((a) => {
      if (provider && bucketOf(a.provider) !== provider) return false
      if (status === 'enabled' && (!a.enabled || isBanned(a))) return false
      if (status === 'disabled' && a.enabled) return false
      if (status === 'banned' && !isBanned(a)) return false
      if (kw) {
        const hay = `${a.email} ${a.nickname} ${a.userId} ${a.provider}`.toLowerCase()
        if (!hay.includes(kw)) return false
      }
      return true
    })
  }, [list, debouncedKeyword, status, provider])

  const pageCount = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const clampedPage = Math.min(page, pageCount - 1)
  const pageRows = filtered.slice(clampedPage * PAGE_SIZE, clampedPage * PAGE_SIZE + PAGE_SIZE)

  function handleRefresh(a: AccountListItem) {
    refreshAccount.mutate(a.id, {
      onSuccess: () => toast.success(t('common.saved')),
      onError: () => toast.error(t('accounts.refreshFailed')),
    })
  }

  function confirmResetQuota() {
    if (!pendingResetQuota) return
    const a = pendingResetQuota
    consumeResetCredit.mutate(
      { id: a.id },
      {
        onSuccess: () => {
          toast.success(t('codex.resetSuccess'))
          setPendingResetQuota(null)
        },
        onError: (err: any) => {
          toast.error(err?.message || t('codex.resetFailed'))
          setPendingResetQuota(null)
        },
      }
    )
  }

  function confirmDelete() {
    if (!pendingDelete) return
    const id = pendingDelete.id
    deleteAccount.mutate(id, {
      onSuccess: () => toast.success(t('accounts.deleteSuccess')),
      onError: () => toast.error(t('common.failed')),
    })
    setPendingDelete(null)
  }

  const selectedIds = [...selected].filter((id) => filtered.some((a) => a.id === id))
  const allSelected = filtered.length > 0 && selectedIds.length === filtered.length

  function runBatch(action: 'enable' | 'disable' | 'refresh') {
    if (selectedIds.length === 0) return
    batch.mutate(
      { ids: selectedIds, action },
      {
        onSuccess: () => toast.success(t('common.saved')),
        onError: () => toast.error(t('common.failed')),
      },
    )
  }

  async function confirmBatchDelete() {
    for (const id of selectedIds) {
      await deleteAccount.mutateAsync(id).catch(() => null)
    }
    toast.success(t('accounts.deleteSuccess'))
    clearAccountSelection()
    setBatchDeleteOpen(false)
  }

  function handleRefreshAll() {
    const allIds = list.map((a) => a.id)
    if (allIds.length === 0) return
    batch.mutate(
      { ids: allIds, action: 'refresh' },
      {
        onSuccess: () => toast.success(t('accounts.refreshAllSuccess') || 'Đã làm mới tất cả tài khoản'),
        onError: () => toast.error(t('accounts.refreshFailed')),
      },
    )
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title={lockedMeta ? t(lockedMeta.labelKey) : t('accounts.title')}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              onClick={handleRefreshAll}
              disabled={batch.isPending || list.length === 0}
              aria-label={t('accounts.refreshAll') || 'Làm mới tất cả'}
            >
              <RotateCw className={`size-4 ${batch.isPending ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">{t('accounts.refreshAll') || 'Làm mới tất cả'}</span>
            </Button>
            <Button variant="outline" size="icon" aria-label={t('theme.status')} onClick={togglePrivacy}>
              {privacy ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </Button>
            <Button variant="outline" onClick={() => setExportOpen(true)} aria-label={t('accounts.export')}>
              <Download className="size-4" />
              <span className="hidden sm:inline">{t('accounts.export')}</span>
            </Button>
            <Button onClick={() => setAddOpen(true)}>
              <Plus className="size-4" />
              {t('accounts.add')}
            </Button>
          </div>
        }
      />

      <AccountFilters hideProvider={!!lockedProvider} />
      <BatchBar
        selectedCount={selectedIds.length}
        allSelected={allSelected}
        onToggleAll={() =>
          allSelected ? clearAccountSelection() : setAccountSelection(filtered.map((a) => a.id))
        }
        onEnable={() => runBatch('enable')}
        onDisable={() => runBatch('disable')}
        onRefresh={() => runBatch('refresh')}
        onDelete={() => setBatchDeleteOpen(true)}
        onClear={clearAccountSelection}
        busy={batch.isPending}
      />

      {accounts.isPending ? (
        <HamsterLoader label={t('detail.loading')} />
      ) : filtered.length === 0 ? (
        <EmptyState title={t('accounts.empty')} />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {pageRows.map((a) => (
              <AccountCard
                key={a.id}
                account={a}
                onTest={setTestTarget}
                onRefresh={handleRefresh}
                onDetail={setDetailTarget}
                onDelete={setPendingDelete}
                onResetQuota={setPendingResetQuota}
                refreshing={refreshAccount.isPending && refreshAccount.variables === a.id}
                resettingQuota={consumeResetCredit.isPending && consumeResetCredit.variables?.id === a.id}
              />
            ))}
          </div>

          {pageCount > 1 && (
            <div className="flex items-center justify-center gap-3 text-sm text-muted-foreground">
              <Button variant="outline" size="sm" disabled={clampedPage <= 0} onClick={() => setPage(clampedPage - 1)}>
                {t('apiKeys.prev')}
              </Button>
              <span>{tp(t, 'apiKeys.pageOf', clampedPage + 1, pageCount)}</span>
              <Button variant="outline" size="sm" disabled={clampedPage >= pageCount - 1} onClick={() => setPage(clampedPage + 1)}>
                {t('apiKeys.next')}
              </Button>
            </div>
          )}
        </>
      )}

      <ConfirmDialog
        open={!!pendingResetQuota}
        onOpenChange={(o) => !o && setPendingResetQuota(null)}
        title={t('codex.confirmResetTitle')}
        description={t('codex.confirmResetDesc')}
        confirmLabel={t('codex.consumeReset')}
        onConfirm={confirmResetQuota}
      />

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title={t('accounts.confirmDelete')}
        confirmLabel={t('accounts.delete')}
        destructive
        onConfirm={confirmDelete}
      />

      <ConfirmDialog
        open={batchDeleteOpen}
        onOpenChange={setBatchDeleteOpen}
        title={t('accounts.confirmDelete')}
        description={tp(t, 'export.selected', selectedIds.length)}
        confirmLabel={t('accounts.delete')}
        destructive
        onConfirm={confirmBatchDelete}
      />

      <AddAccountDialog open={addOpen} onOpenChange={setAddOpen} provider={lockedProvider} />

      <AccountDetailDialog
        account={detailTarget}
        onClose={() => setDetailTarget(null)}
      />

      <TestAccountDialog
        account={testTarget}
        onClose={() => setTestTarget(null)}
      />

      <ExportDialog
        open={exportOpen}
        onOpenChange={setExportOpen}
        accounts={list}
      />
    </div>
  )
}
