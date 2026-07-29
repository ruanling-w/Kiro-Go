// Logs — live SSE stream + xterm renderer. Filters (status / per-API-key /
// search) run client-side over the live buffer. The Live switch pauses/resumes
// the SSE connection. Clear goes through the shared ConfirmDialog and also
// empties the local buffer immediately.
import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Trash2, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useLogStream } from '@/hooks/queries/useLogStream'
import { useClearLogs } from '@/hooks/mutations/useLogMutations'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import { PageHeader } from '@/components/shared/PageHeader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { LogsTerminal } from './components/LogsTerminal'

type StatusFilter = 'all' | 'success' | 'error'

export default function LogsPage() {
  const { t } = useTranslation()
  const [params, setParams] = useSearchParams()
  const [live, setLive] = useState(true)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [search, setSearch] = useState('')
  const [clearOpen, setClearOpen] = useState(false)

  const stream = useLogStream(!live)
  const clearLogs = useClearLogs()
  const apiKeys = useApiKeys()

  const apiKeyFilter = params.get('apiKey') ?? ''

  const setApiKeyFilter = (id: string) => {
    const next = new URLSearchParams(params)
    if (id) next.set('apiKey', id)
    else next.delete('apiKey')
    setParams(next, { replace: true })
  }

  const keyNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const k of apiKeys.data ?? []) map.set(k.id, k.name || t('apiKeys.unnamed'))
    return map
  }, [apiKeys.data, t])

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    return stream.logs.filter((log) => {
      if (statusFilter === 'success' && log.status !== 'success') return false
      if (statusFilter === 'error' && log.status === 'success') return false
      if (apiKeyFilter && log.apiKeyId !== apiKeyFilter) return false
      if (term) {
        const keyName = log.apiKeyId ? keyNames.get(log.apiKeyId) ?? '' : ''
        const haystack = [
          log.model,
          log.endpoint,
          log.provider,
          log.accountId,
          log.clientIp,
          log.errorType,
          log.error,
          keyName,
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase()
        if (!haystack.includes(term)) return false
      }
      return true
    })
  }, [stream.logs, statusFilter, apiKeyFilter, search, keyNames])

  const statusBadge = {
    connecting: { variant: 'secondary' as const, label: t('logs.reconnecting') },
    live: { variant: 'default' as const, label: t('logs.live') },
    error: { variant: 'destructive' as const, label: t('logs.connectionLost') },
  }[live ? stream.status : ('paused' as never)] ?? {
    variant: 'outline' as const,
    label: t('logs.paused'),
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title={t('logs.title')}
        actions={
          <div className="flex items-center gap-2">
            <Badge variant={statusBadge.variant}>{statusBadge.label}</Badge>
            <Button variant="destructive" size="sm" onClick={() => setClearOpen(true)}>
              <Trash2 className="size-4" />
              {t('logs.clear')}
            </Button>
          </div>
        }
      />

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-48 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t('filter.search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
          <SelectTrigger className="w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('logs.filterAll')}</SelectItem>
            <SelectItem value="success">{t('logs.filterSuccess')}</SelectItem>
            <SelectItem value="error">{t('logs.filterError')}</SelectItem>
          </SelectContent>
        </Select>

        <Select value={apiKeyFilter || 'all'} onValueChange={(v) => setApiKeyFilter(v === 'all' ? '' : v)}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder={t('logs.apiKey')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('logs.apiKey')}</SelectItem>
            {(apiKeys.data ?? []).map((k) => (
              <SelectItem key={k.id} value={k.id}>
                {k.name || t('apiKeys.unnamed')} · {k.keyMasked}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="flex items-center gap-2">
          <Switch id="live" checked={live} onCheckedChange={setLive} />
          <Label htmlFor="live" className="text-sm">{t('logs.autoRefresh')}</Label>
        </div>
      </div>

      <LogsTerminal logs={filtered} keyNames={keyNames} searchTerm={search} />

      <ConfirmDialog
        open={clearOpen}
        onOpenChange={setClearOpen}
        title={t('logs.clearConfirm')}
        confirmLabel={t('logs.clear')}
        destructive
        onConfirm={() =>
          clearLogs.mutate(undefined, {
            onSuccess: () => {
              stream.clear()
              toast.success(t('logs.cleared'))
            },
            onError: () => toast.error(t('common.failed')),
          })
        }
      />
    </div>
  )
}
