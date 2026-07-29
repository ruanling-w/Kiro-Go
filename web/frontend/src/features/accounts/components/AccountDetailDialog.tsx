// AccountDetailDialog — edit an account's nickname/weight/machineId/proxy + view
// its full detail (secrets from GET /full) and toggle overage. Loads the full
// account on open; saves via useUpdateAccount; overage via useSetAccountOverage.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { AccountListItem } from '@/types/account'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { CodexQuota } from './CodexQuota'
import { bucketOf } from '@/config/providers'
import { useAccountFull } from '@/hooks/queries/useAccounts'
import {
  useUpdateAccount,
  useSetAccountOverage,
} from '@/hooks/mutations/useAccountMutations'
import { generateMachineId } from '@/services/accounts.service'

interface Props {
  account: AccountListItem | null
  onClose: () => void
}

export function AccountDetailDialog({ account, onClose }: Props) {
  const { t } = useTranslation()
  const full = useAccountFull(account?.id ?? null)
  const update = useUpdateAccount()
  const overage = useSetAccountOverage()

  const [nickname, setNickname] = useState('')
  const [weight, setWeight] = useState(1)
  const [machineId, setMachineId] = useState('')
  const [proxyURL, setProxyURL] = useState('')

  useEffect(() => {
    if (account) {
      setNickname(account.nickname || '')
      setWeight(account.weight || 1)
      setMachineId(account.machineId || '')
      setProxyURL(account.proxyURL || '')
    }
  }, [account])

  function save() {
    if (!account) return
    update.mutate(
      { id: account.id, patch: { nickname, weight, machineId, proxyURL } },
      {
        onSuccess: () => {
          toast.success(t('detail.saved'))
          onClose()
        },
        onError: () => toast.error(t('detail.saveFailed')),
      },
    )
  }

  function toggleOverage(enabled: boolean) {
    if (!account) return
    overage.mutate(
      { id: account.id, enabled },
      {
        onSuccess: () => toast.success(t('detail.saved')),
        onError: () => toast.error(t('accounts.overageSwitchFailed')),
      },
    )
  }

  async function regenMachineId() {
    try {
      const { machineId: id } = await generateMachineId()
      setMachineId(id)
    } catch {
      toast.error(t('detail.generateFailed'))
    }
  }

  const detailAccount = full.data ?? account
  const overageOn = (detailAccount?.overageStatus ?? account?.overageStatus) === 'enabled'
  const overageCapable = (detailAccount?.overageCapability ?? account?.overageCapability) !== 'none'
  const isCodex = detailAccount
    ? bucketOf(detailAccount.provider) === 'codex' || detailAccount.authMethod?.toLowerCase() === 'codex'
    : false

  return (
    <Dialog open={!!account} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[85vh] max-w-lg overflow-auto">
        <DialogHeader>
          <DialogTitle>{t('accounts.detail')}</DialogTitle>
        </DialogHeader>

        {full.isPending ? (
          <HamsterLoader size="sm" label={t('detail.loading')} />
        ) : (
          <div className="space-y-4">
            <Field label={t('detail.email')}>
              <p className="truncate text-sm text-muted-foreground">{account?.email || account?.id}</p>
            </Field>

            {isCodex && detailAccount && (
              <CodexQuota
                windows={detailAccount.codexQuota}
                limitReached={detailAccount.codexLimitReached}
                resetCredits={detailAccount.codexResetCredits}
                detail
              />
            )}

            <div className="space-y-2">
              <Label htmlFor="nickname">{t('detail.nickname')}</Label>
              <Input id="nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} />
            </div>

            <div className="space-y-2">
              <Label htmlFor="weight">{t('detail.weight')}</Label>
              <Input
                id="weight"
                type="number"
                min={0}
                value={weight}
                onChange={(e) => setWeight(Number(e.target.value))}
              />
              <p className="text-xs text-muted-foreground">{t('detail.weightHint')}</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="machineId">{t('detail.machineId')}</Label>
              <div className="flex items-center gap-2">
                <Input id="machineId" value={machineId} onChange={(e) => setMachineId(e.target.value)} />
                <Button variant="outline" size="sm" onClick={regenMachineId}>
                  {t('detail.generate')}
                </Button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="proxy">{t('detail.proxyURL')}</Label>
              <Input
                id="proxy"
                value={proxyURL}
                onChange={(e) => setProxyURL(e.target.value)}
                placeholder="socks5://user:pass@host:port"
              />
              <p className="text-xs text-muted-foreground">{t('detail.proxyHint')}</p>
            </div>

            {overageCapable && (
              <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
                <div>
                  <Label>{t('detail.overage')}</Label>
                  <p className="mt-1 text-xs text-muted-foreground">{t('detail.overageHint')}</p>
                </div>
                <Switch checked={overageOn} disabled={overage.isPending} onCheckedChange={toggleOverage} />
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={save} disabled={update.isPending}>
            {t('detail.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      {children}
    </div>
  )
}
