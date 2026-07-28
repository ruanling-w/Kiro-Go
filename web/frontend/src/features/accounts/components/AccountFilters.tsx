// AccountFilters — search box (debounced upstream) + status filter + provider
// filter. All state lives in uiStore so it survives navigation from the
// Providers landing (which pre-sets providerFilter).
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useUiStore, type AccountStatusFilter } from '@/stores/uiStore'
import { PROVIDERS } from '@/config/providers'

const STATUS_OPTIONS: { value: AccountStatusFilter; key: string }[] = [
  { value: 'all', key: 'filter.all' },
  { value: 'enabled', key: 'filter.enabled' },
  { value: 'disabled', key: 'filter.disabled' },
  { value: 'banned', key: 'filter.banned' },
]

export function AccountFilters({ hideProvider }: { hideProvider?: boolean }) {
  const { t } = useTranslation()
  const keyword = useUiStore((s) => s.accountKeyword)
  const setKeyword = useUiStore((s) => s.setAccountKeyword)
  const status = useUiStore((s) => s.accountStatus)
  const setStatus = useUiStore((s) => s.setAccountStatus)
  const provider = useUiStore((s) => s.providerFilter)
  const setProvider = useUiStore((s) => s.setProviderFilter)

  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative min-w-56 flex-1">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          className="pl-9"
          placeholder={t('filter.search')}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
      </div>

      <Select value={status} onValueChange={(v) => setStatus(v as AccountStatusFilter)}>
        <SelectTrigger className="w-40">
          <SelectValue placeholder={t('filter.status')} />
        </SelectTrigger>
        <SelectContent>
          {STATUS_OPTIONS.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {t(o.key)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {!hideProvider && (
        <Select value={provider || 'all'} onValueChange={(v) => setProvider(v === 'all' ? '' : v)}>
          <SelectTrigger className="w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('filter.all')}</SelectItem>
            {PROVIDERS.map((p) => (
              <SelectItem key={p.key} value={p.key}>
                {t(p.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    </div>
  )
}
