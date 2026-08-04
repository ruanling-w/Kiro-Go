// ApiKeyIpsDialog — the unique IPs that have used a key, with per-IP request
// count, live RPM, and last-seen. Opens when `keyId` is non-null.
import { useTranslation } from 'react-i18next'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { EmptyState } from '@/components/shared/EmptyState'
import { useApiKeyIPs } from '@/hooks/queries/useApiKeys'
import { formatNumber, formatUnixSeconds } from '@/lib/format'

interface Props {
  keyId: string | null
  onClose: () => void
  /** Optional live limit from the parent row for header context. */
  rpmLimit?: number
}

export function ApiKeyIpsDialog({ keyId, onClose, rpmLimit = 0 }: Props) {
  const { t } = useTranslation()
  const query = useApiKeyIPs(keyId)
  const ips = query.data?.ips ?? []
  const rpm = query.data?.rpm ?? 0

  return (
    <Dialog open={!!keyId} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2">
            <span>{t('apiKeys.ipsTitle')}</span>
            <Badge variant="secondary" className="font-mono tabular-nums">
              {t('apiKeys.rpm')}:{' '}
              {rpmLimit > 0
                ? `${formatNumber(rpm)} / ${formatNumber(rpmLimit)}`
                : formatNumber(rpm)}
            </Badge>
          </DialogTitle>
        </DialogHeader>

        <div className="max-h-[60vh] overflow-auto">
          {query.isPending ? (
            <HamsterLoader label={t('detail.loading')} />
          ) : ips.length === 0 ? (
            <EmptyState title={t('apiKeys.ipsEmpty')} />
          ) : (
            <ul className="divide-y">
              {ips.map((ip) => (
                <li
                  key={ip.ip}
                  className="flex items-center justify-between gap-3 py-2.5 text-sm"
                >
                  <code className="font-mono">{ip.ip}</code>
                  <div className="flex flex-wrap justify-end gap-3 text-muted-foreground">
                    <span className="tabular-nums">
                      {formatNumber(ip.requests)} · {t('apiKeys.ipsRequests')}
                    </span>
                    <span className="tabular-nums">
                      {formatNumber(ip.rpm ?? 0)} {t('apiKeys.rpm')}
                    </span>
                    <span className="whitespace-nowrap">
                      {formatUnixSeconds(ip.lastSeen, {
                        dateStyle: 'short',
                        timeStyle: 'short',
                      })}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
