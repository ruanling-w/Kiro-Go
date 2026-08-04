// ApiKeyTable — the keys list via the shared DataTable (sort + paging). Search +
// status filter are applied upstream by the page (uiStore). Each row exposes
// reveal/copy, toggle-enable, edit, reset-usage, view-IPs, delete.
import { useTranslation } from 'react-i18next'
import { Pencil, RotateCcw, Trash2, Network, ScrollText } from 'lucide-react'
import type { ApiKeyView } from '@/types/apikey'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { formatNumber, formatUnixSeconds } from '@/lib/format'
import { ApiKeyRevealCell } from './ApiKeyRevealCell'

interface Props {
  rows: ApiKeyView[]
  onEdit: (k: ApiKeyView) => void
  onDelete: (k: ApiKeyView) => void
  onReset: (k: ApiKeyView) => void
  onToggle: (k: ApiKeyView, enabled: boolean) => void
  onViewIps: (k: ApiKeyView) => void
  onViewLogs: (k: ApiKeyView) => void
}

function keyStatus(k: ApiKeyView): { tone: 'success' | 'neutral' | 'danger'; key: string } {
  if (k.expired) return { tone: 'danger', key: 'apiKeys.expired' }
  if (!k.enabled) return { tone: 'neutral', key: 'apiKeys.disabled' }
  return { tone: 'success', key: 'apiKeys.statusEnabled' }
}

export function ApiKeyTable({ rows, onEdit, onDelete, onReset, onToggle, onViewIps, onViewLogs }: Props) {
  const { t } = useTranslation()

  const columns: Column<ApiKeyView>[] = [
    {
      id: 'name',
      header: t('apiKeys.colName'),
      mobileRole: 'primary',
      sortValue: (k) => k.name || '',
      cell: (k) => (
        <div className="min-w-0">
          <p className="truncate font-medium">{k.name || t('apiKeys.unnamed')}</p>
          {k.migrated && (
            <span className="text-[11px] text-muted-foreground">{t('apiKeys.migrated')}</span>
          )}
        </div>
      ),
    },
    {
      id: 'key',
      header: t('apiKeys.colKey'),
      mobileRole: 'secondary',
      cell: (k) => <ApiKeyRevealCell id={k.id} masked={k.keyMasked} />,
    },
    {
      id: 'usage',
      header: t('apiKeys.requests'),
      mobileRole: 'meta',
      align: 'right',
      sortValue: (k) => k.requestsCount,
      cell: (k) => <span className="tabular-nums">{formatNumber(k.requestsCount)}</span>,
    },
    {
      id: 'tokens',
      header: t('apiKeys.tokens'),
      mobileRole: 'meta',
      align: 'right',
      sortValue: (k) => k.tokensUsed,
      cell: (k) => <span className="tabular-nums">{formatNumber(k.tokensUsed)}</span>,
    },
    {
      id: 'ips',
      header: t('apiKeys.colIPs'),
      mobileRole: 'meta',
      align: 'right',
      sortValue: (k) => k.uniqueIps,
      cell: (k) => (
        <button
          type="button"
          className="tabular-nums hover:text-foreground hover:underline"
          onClick={() => onViewIps(k)}
        >
          {formatNumber(k.uniqueIps)}
        </button>
      ),
    },
    {
      id: 'created',
      header: t('apiKeys.created'),
      mobileRole: 'meta',
      align: 'right',
      sortValue: (k) => k.createdAt,
      cell: (k) => (
        <span className="whitespace-nowrap text-muted-foreground">
          {formatUnixSeconds(k.createdAt, { dateStyle: 'short' })}
        </span>
      ),
    },
    {
      id: 'status',
      header: t('apiKeys.colStatus'),
      mobileRole: 'badge',
      align: 'center',
      cell: (k) => {
        const s = keyStatus(k)
        return <StatusBadge tone={s.tone}>{t(s.key)}</StatusBadge>
      },
    },
    {
      id: 'actions',
      header: t('apiKeys.colActions'),
      mobileRole: 'actions',
      align: 'right',
      cell: (k) => (
        <div className="flex w-full flex-wrap items-center justify-between gap-1">
          <Switch
            checked={k.enabled}
            onCheckedChange={(v) => onToggle(k, v)}
            aria-label={t('apiKeys.statusEnabled')}
          />
          <div className="flex flex-wrap items-center justify-end gap-0.5">
            <Button variant="ghost" size="icon-sm" onClick={() => onViewLogs(k)} aria-label={t('logs.title')}>
              <ScrollText className="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={() => onViewIps(k)} aria-label={t('apiKeys.viewIPs')}>
              <Network className="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={() => onEdit(k)} aria-label={t('apiKeys.actionEdit')}>
              <Pencil className="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={() => onReset(k)} aria-label={t('apiKeys.actionReset')}>
              <RotateCcw className="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" onClick={() => onDelete(k)} aria-label={t('apiKeys.actionDelete')}>
              <Trash2 className="size-4 text-destructive" />
            </Button>
          </div>
        </div>
      ),
    },
  ]

  return (
    <DataTable
      rows={rows}
      columns={columns}
      rowKey={(k) => k.id}
      emptyMessage={t('apiKeys.noMatches')}
      initialSort={{ id: 'created', dir: 'desc' }}
    />
  )
}
