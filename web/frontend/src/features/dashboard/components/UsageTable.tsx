// UsageTable — per-account token/credit/request usage (merged from the old Usage
// view). Uses the shared DataTable for sort + pagination; search filters
// client-side by email/nickname/provider. Totals footer sums the filtered set.
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import type { AccountListItem } from '@/types/account'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { Input } from '@/components/ui/input'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { useDebounce } from '@/hooks/useDebounce'
import { useUiStore } from '@/stores/uiStore'
import { maskEmail } from '@/lib/mask'
import { formatNumber } from '@/lib/format'
import { bucketOf } from '@/config/providers'

interface UsageTableProps {
  accounts: AccountListItem[]
  loading: boolean
}

export function UsageTable({ accounts, loading }: UsageTableProps) {
  const { t } = useTranslation()
  const privacy = useUiStore((s) => s.privacyMode)
  const [keyword, setKeyword] = useState('')
  const debounced = useDebounce(keyword, 300)

  const rows = useMemo(() => {
    const kw = debounced.trim().toLowerCase()
    if (!kw) return accounts
    return accounts.filter((a) =>
      [a.email, a.nickname, a.provider].some((v) => v?.toLowerCase().includes(kw)),
    )
  }, [accounts, debounced])

  const totals = useMemo(
    () =>
      rows.reduce(
        (acc, a) => {
          acc.requests += a.requestCount || 0
          acc.tokens += a.totalTokens || 0
          acc.credits += a.totalCredits || 0
          return acc
        },
        { requests: 0, tokens: 0, credits: 0 },
      ),
    [rows],
  )

  const columns: Column<AccountListItem>[] = [
    {
      id: 'account',
      header: t('usage.account'),
      mobileRole: 'primary',
      cell: (a) => (
        <div className="flex min-w-0 items-center gap-2">
          <ProviderIcon provider={bucketOf(a.provider)} className="size-4 shrink-0" />
          <span className="truncate">{privacy ? maskEmail(a.email) : a.email || a.id}</span>
        </div>
      ),
      sortValue: (a) => a.email || a.id,
    },
    {
      id: 'requests',
      header: t('usage.requests'),
      mobileRole: 'meta',
      align: 'right',
      cell: (a) => <span className="tabular-nums">{formatNumber(a.requestCount)}</span>,
      sortValue: (a) => a.requestCount,
    },
    {
      id: 'tokens',
      header: t('usage.tokens'),
      mobileRole: 'meta',
      align: 'right',
      cell: (a) => <span className="tabular-nums">{formatNumber(a.totalTokens)}</span>,
      sortValue: (a) => a.totalTokens,
    },
    {
      id: 'credits',
      header: t('usage.credits'),
      mobileRole: 'meta',
      align: 'right',
      cell: (a) => <span className="tabular-nums">{formatNumber(a.totalCredits)}</span>,
      sortValue: (a) => a.totalCredits,
    },
  ]

  if (loading) return <HamsterLoader label={t('detail.loading')} />
  if (accounts.length === 0) return <EmptyState message={t('accounts.empty')} />

  return (
    <div className="space-y-3">
      <div className="relative w-full max-w-full sm:max-w-xs">
        <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          placeholder={t('filter.search')}
          className="pl-8"
        />
      </div>
      <DataTable
        rows={rows}
        columns={columns}
        rowKey={(a) => a.id}
        initialSort={{ id: 'tokens', dir: 'desc' }}
        footer={
          <span className="mr-2 text-xs">
            {t('usage.total')}: {formatNumber(totals.tokens)} · {formatNumber(totals.credits)} ·{' '}
            {formatNumber(totals.requests)}
          </span>
        }
      />
    </div>
  )
}
