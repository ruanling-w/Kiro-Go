// SecurityCard — trust-proxy toggle + blocked-IP list (add/unblock). Blocked IPs
// come from useBlockedIps; trust-proxy from useSecuritySettings.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ShieldX, Plus } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { formatUnixSeconds } from '@/lib/format'
import { useSecuritySettings, useBlockedIps } from '@/hooks/queries/useSecurity'
import {
  useUpdateSecuritySettings,
  useBlockIp,
  useUnblockIp,
} from '@/hooks/mutations/useSecurityMutations'
import { toast } from 'sonner'

export function SecurityCard() {
  const { t } = useTranslation()
  const settings = useSecuritySettings()
  const blocked = useBlockedIps()
  const updateSettings = useUpdateSecuritySettings()
  const blockIp = useBlockIp()
  const unblockIp = useUnblockIp()

  const [ip, setIp] = useState('')
  const [reason, setReason] = useState('')

  function handleBlock() {
    const v = ip.trim()
    if (!v) return
    blockIp.mutate(
      { ip: v, reason: reason.trim() || undefined },
      {
        onSuccess: () => {
          toast.success(t('security.blocked'))
          setIp('')
          setReason('')
        },
        onError: () => toast.error(t('security.invalidIP')),
      },
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('security.title')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between gap-4">
          <div>
            <Label>{t('security.trustProxy')}</Label>
            <p className="text-sm text-muted-foreground">{t('security.trustProxyHint')}</p>
          </div>
          <Switch
            checked={settings.data?.trustProxyHeaders ?? false}
            onCheckedChange={(v) =>
              updateSettings.mutate(
                { trustProxyHeaders: v },
                { onSuccess: () => toast.success(t('security.saved')) },
              )
            }
          />
        </div>

        <div className="space-y-2">
          <Label>{t('security.blockedTitle')}</Label>
          <div className="flex flex-wrap gap-2">
            <Input
              className="min-w-40 flex-1"
              placeholder={t('security.ipPlaceholder')}
              value={ip}
              onChange={(e) => setIp(e.target.value)}
            />
            <Input
              className="min-w-40 flex-1"
              placeholder={t('security.reasonPlaceholder')}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            <Button onClick={handleBlock} disabled={blockIp.isPending || !ip.trim()}>
              <Plus className="size-4" />
              {t('security.blockIP')}
            </Button>
          </div>

          {(blocked.data?.length ?? 0) === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              {t('security.blockedEmpty')}
            </p>
          ) : (
            <ul className="divide-y rounded-lg border">
              {blocked.data!.map((b) => (
                <li key={b.ip} className="flex items-center justify-between gap-3 px-3 py-2">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-sm">{b.ip}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {b.reason || '—'} · {formatUnixSeconds(b.blockedAt, { dateStyle: 'short', timeStyle: 'short' })}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      unblockIp.mutate(b.ip, {
                        onSuccess: () => toast.success(t('security.unblocked')),
                      })
                    }
                  >
                    <ShieldX className="size-4" />
                    {t('security.unblock')}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
