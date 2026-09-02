// Logs — live SSE stream + 9router-style mono table. Filters (status /
// per-API-key / search) run client-side over the live buffer. The Live switch
// pauses/resumes the SSE connection. Clear goes through ConfirmDialog and also
// empties the local buffer immediately.
import { useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Trash2, Search } from 'lucide-react'
import { toast } from 'sonner'
import { useLogStream } from '@/hooks/queries/useLogStream'
import { useClearLogs } from '@/hooks/mutations/useLogMutations'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import { useAccounts } from '@/hooks/queries/useAccounts'
import { useUiStore } from '@/stores/uiStore'
import { displayEmail } from '@/lib/mask'
import { formatNumber } from '@/lib/format'
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
import { Card } from '@/components/ui/card'
import { sortModelTokenSummary, summarizeModelTokens } from './modelTokenSummary'
import { LogsTable } from './components/LogsTable'

type StatusFilter = 'all' | 'success' | 'error'

export default function LogsPage() {
  const { t } = useTranslation()
  const [params, setParams] = useSearchParams()
  const [live, setLive] = useState(true)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [search, setSearch] = useState('')
  const [modelFilter, setModelFilter] = useState('')
  const [summarySort, setSummarySort] = useState<'total-desc' | 'total-asc' | 'avg-desc' | 'avg-asc' | 'requests-desc' | 'model-asc'>('total-desc')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [clearOpen, setClearOpen] = useState(false)

  const stream = useLogStream(!live)
  const clearLogs = useClearLogs()
  const apiKeys = useApiKeys()
  const accounts = useAccounts()
  const privacy = useUiStore((s) => s.privacyMode)

  const apiKeyFilter = params.get('apiKey') ?? ''

  const setApiKeyFilter = (id: string) => {
    const next = new URLSearchParams(params)
    if (id) next.set('apiKey', id)
    else next.delete('apiKey')
    setParams(next, { replace: true })
  }

  // Resolve opaque ids → human labels for the table + search haystack.
  const keyNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const k of apiKeys.data ?? []) {
      const name = k.name?.trim() || t('apiKeys.unnamed')
      map.set(k.id, k.keyMasked ? `${name} · ${k.keyMasked}` : name)
    }
    return map
  }, [apiKeys.data, t])

  const accountNames = useMemo(() => {
    const map = new Map<string, string>()
    for (const a of accounts.data ?? []) {
      const email = a.email ? displayEmail(a.email, a.id, privacy) : ''
      const nickname = a.nickname?.trim()
      let label = ''
      if (nickname && email) {
        label = `${nickname} (${email})`
      } else if (nickname) {
        label = nickname
      } else if (email) {
        label = email
      } else {
        label = a.userId || a.id
      }
      map.set(a.id, label)
    }
    return map
  }, [accounts.data, privacy])

  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase()
    const modelTerm = modelFilter.trim().toLowerCase()
    return stream.logs.filter((log) => {
      if (statusFilter === 'success' && log.status !== 'success') return false
      if (statusFilter === 'error' && log.status === 'success') return false
      if (apiKeyFilter && log.apiKeyId !== apiKeyFilter) return false
      if (startDate && log.time < Math.floor(new Date(`${startDate}T00:00:00`).getTime() / 1000)) return false
      if (endDate && log.time > Math.floor(new Date(`${endDate}T23:59:59.999`).getTime() / 1000)) return false
      if (modelTerm && !String(log.model ?? '').toLowerCase().includes(modelTerm)) return false
      if (term) {
        const keyName = log.apiKeyId ? keyNames.get(log.apiKeyId) ?? '' : ''
        const accountName = log.accountId ? accountNames.get(log.accountId) ?? '' : ''
        const haystack = [
          log.model,
          log.endpoint,
          log.provider,
          log.accountId,
          accountName,
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
  }, [stream.logs, statusFilter, apiKeyFilter, search, modelFilter, startDate, endDate, keyNames, accountNames])

  const modelSummary = useMemo(() => {
    const rows = summarizeModelTokens(filtered, startDate, endDate)
    return sortModelTokenSummary(rows, summarySort)
  }, [filtered, startDate, endDate, summarySort])

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
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={statusBadge.variant}>{statusBadge.label}</Badge>
            <Button variant="destructive" size="sm" onClick={() => setClearOpen(true)}>
              <Trash2 className="size-4" />
              {t('logs.clear')}
            </Button>
          </div>
        }
      />

      <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
        <div className="relative min-w-0 w-full flex-1 sm:min-w-48">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t('filter.search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
            <SelectTrigger className="w-full sm:w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t('logs.filterAll')}</SelectItem>
              <SelectItem value="success">{t('logs.filterSuccess')}</SelectItem>
              <SelectItem value="error">{t('logs.filterError')}</SelectItem>
            </SelectContent>
          </Select>

          <Select
            value={apiKeyFilter || 'all'}
            onValueChange={(v) => setApiKeyFilter(v === 'all' ? '' : v)}
          >
            <SelectTrigger className="w-full sm:w-52">
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
            <label className="text-xs text-muted-foreground">Model</label>
            <Input
              value={modelFilter}
              onChange={(e) => setModelFilter(e.target.value)}
              placeholder="e.g. gpt-5.6-luna"
              className="w-[180px]"
            />
          </div>

          <div className="flex items-center gap-2">
            <label className="text-xs text-muted-foreground">From</label>
            <Input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="w-[150px]" />
          </div>

          <div className="flex items-center gap-2">
            <label className="text-xs text-muted-foreground">To</label>
            <Input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="w-[150px]" />
          </div>

          <div className="flex items-center gap-2">
            <Switch id="live" checked={live} onCheckedChange={setLive} />
            <Label htmlFor="live" className="text-sm">
              {t('logs.autoRefresh')}
            </Label>
          </div>
        </div>
      </div>

      <Card className="p-3">
        <div className="mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h3 className="text-sm font-medium">Model token summary</h3>
          <div className="flex items-center gap-2">
            <label className="text-xs text-muted-foreground">Sort</label>
            <Select value={summarySort} onValueChange={(v) => setSummarySort(v as typeof summarySort)}>
              <SelectTrigger className="w-[180px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="total-desc">Total tokens ↓</SelectItem>
                <SelectItem value="total-asc">Total tokens ↑</SelectItem>
                <SelectItem value="avg-desc">Avg tokens ↓</SelectItem>
                <SelectItem value="avg-asc">Avg tokens ↑</SelectItem>
                <SelectItem value="requests-desc">Requests ↓</SelectItem>
                <SelectItem value="model-asc">Model A–Z</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        {(startDate || endDate) && (
          <div className="mb-2 text-xs text-muted-foreground">
            {startDate || '—'} → {endDate || '—'}
          </div>
        )}
        <div className="overflow-auto">
          <table className="w-full border-collapse text-left font-mono text-xs">
            <thead>
              <tr className="border-b text-muted-foreground">
                <th className="px-2 py-1.5 text-left">Model</th>
                <th className="px-2 py-1.5 text-right">Requests</th>
                <th className="px-2 py-1.5 text-right">Total</th>
                <th className="px-2 py-1.5 text-right">Cache read</th>
                <th className="px-2 py-1.5 text-right">Cache write</th>
              </tr>
            </thead>
            <tbody>
              {modelSummary.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-2 py-3 text-muted-foreground">No model data for this range</td>
                </tr>
              ) : (
                modelSummary.map((row) => (
                  <tr key={row.model} className="border-b border-border/40">
                    <td className="px-2 py-1.5 font-medium">{row.model}</td>
                    <td className="px-2 py-1.5 text-right">{formatNumber(row.requests)}</td>
                    <td className="px-2 py-1.5 text-right">{formatNumber(row.totalTokens)}</td>
                    <td className="px-2 py-1.5 text-right">{formatNumber(row.cacheReadTokens)}</td>
                    <td className="px-2 py-1.5 text-right">{formatNumber(row.cacheCreationTokens)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <LogsTable
        logs={filtered}
        keyNames={keyNames}
        accountNames={accountNames}
        totalCount={stream.logs.length}
      />

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
