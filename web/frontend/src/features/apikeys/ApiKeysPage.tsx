// API Keys — three stacked cards: API-access toggle, security (blocked IPs), and
// the keys table with full CRUD + search/filter/paging. "View logs" on a row
// deep-links to /logs?apiKey=<id>. Reveal/reset/edit/delete go through mutations
// + the shared ConfirmDialog.
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, Search } from 'lucide-react'
import { toast } from 'sonner'
import type { ApiKeyView } from '@/types/apikey'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import {
  useDeleteApiKey,
  useResetApiKeyUsage,
  useUpdateApiKey,
} from '@/hooks/mutations/useApiKeyMutations'
import { PageHeader } from '@/components/shared/PageHeader'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useUiStore, type ApiKeyStatusFilter } from '@/stores/uiStore'
import { useDebounce } from '@/hooks/useDebounce'
import { ApiKeyTable } from './components/ApiKeyTable'
import { ApiAccessCard } from './components/ApiAccessCard'
import { SecurityCard } from './components/SecurityCard'
import { ApiKeyFormDialog } from './components/ApiKeyFormDialog'
import { ApiKeyIpsDialog } from './components/ApiKeyIpsDialog'

const STATUS_OPTIONS: { value: ApiKeyStatusFilter; key: string }[] = [
  { value: 'all', key: 'apiKeys.filterAll' },
  { value: 'active', key: 'apiKeys.filterEnabled' },
  { value: 'disabled', key: 'apiKeys.filterDisabled' },
  { value: 'expired', key: 'apiKeys.filterExpired' },
]

export default function ApiKeysPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const apiKeys = useApiKeys()
  const updateKey = useUpdateApiKey()
  const deleteKey = useDeleteApiKey()
  const resetKey = useResetApiKeyUsage()

  const keyword = useUiStore((s) => s.apiKeyKeyword)
  const setKeyword = useUiStore((s) => s.setApiKeyKeyword)
  const status = useUiStore((s) => s.apiKeyStatus)
  const setStatus = useUiStore((s) => s.setApiKeyStatus)
  const debounced = useDebounce(keyword, 300)

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ApiKeyView | null>(null)
  const [pendingDelete, setPendingDelete] = useState<ApiKeyView | null>(null)
  const [pendingReset, setPendingReset] = useState<ApiKeyView | null>(null)
  const [ipsFor, setIpsFor] = useState<string | null>(null)

  const list = apiKeys.data ?? []
  const hasEnabledKey = list.some((k) => k.enabled && !k.expired)

  const filtered = useMemo(() => {
    const kw = debounced.trim().toLowerCase()
    return list.filter((k) => {
      if (status === 'active' && (!k.enabled || k.expired)) return false
      if (status === 'disabled' && k.enabled) return false
      if (status === 'expired' && !k.expired) return false
      if (kw) {
        const hay = `${k.name} ${k.keyMasked}`.toLowerCase()
        if (!hay.includes(kw)) return false
      }
      return true
    })
  }, [list, debounced, status])

  function handleToggle(k: ApiKeyView, enabled: boolean) {
    updateKey.mutate(
      { id: k.id, patch: { enabled } },
      { onError: () => toast.error(t('common.failed')) },
    )
  }

  function confirmDelete() {
    if (!pendingDelete) return
    deleteKey.mutate(pendingDelete.id, {
      onSuccess: () => toast.success(t('apiKeys.deleteSuccess')),
      onError: () => toast.error(t('common.failed')),
    })
    setPendingDelete(null)
  }

  function confirmReset() {
    if (!pendingReset) return
    resetKey.mutate(pendingReset.id, {
      onSuccess: () => toast.success(t('apiKeys.usageReset')),
      onError: () => toast.error(t('common.failed')),
    })
    setPendingReset(null)
  }

  function openEdit(k: ApiKeyView) {
    setEditing(k)
    setFormOpen(true)
  }

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  return (
    <div className="space-y-5">
      <PageHeader title={t('nav.apikeys')} />

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ApiAccessCard hasEnabledKey={hasEnabledKey} />
        <SecurityCard />
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <CardTitle>{t('apiKeys.listTitle')}</CardTitle>
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            {t('apiKeys.add')}
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder={t('apiKeys.searchPlaceholder')}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
            </div>
            <Select value={status} onValueChange={(v) => setStatus(v as ApiKeyStatusFilter)}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder={t('apiKeys.filterStatus')} />
              </SelectTrigger>
              <SelectContent>
                {STATUS_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {t(o.key)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {apiKeys.isPending ? (
            <HamsterLoader label={t('detail.loading')} />
          ) : (
            <ApiKeyTable
              rows={filtered}
              onEdit={openEdit}
              onDelete={setPendingDelete}
              onReset={setPendingReset}
              onToggle={handleToggle}
              onViewIps={(k) => setIpsFor(k.id)}
              onViewLogs={(k) => navigate(`/logs?apiKey=${encodeURIComponent(k.id)}`)}
            />
          )}
        </CardContent>
      </Card>

      <ApiKeyFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        editing={editing}
      />

      <ApiKeyIpsDialog keyId={ipsFor} onClose={() => setIpsFor(null)} />

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title={t('apiKeys.confirmDelete')}
        confirmLabel={t('apiKeys.actionDelete')}
        destructive
        onConfirm={confirmDelete}
      />

      <ConfirmDialog
        open={!!pendingReset}
        onOpenChange={(o) => !o && setPendingReset(null)}
        title={t('apiKeys.confirmReset')}
        confirmLabel={t('apiKeys.actionReset')}
        onConfirm={confirmReset}
      />
    </div>
  )
}
