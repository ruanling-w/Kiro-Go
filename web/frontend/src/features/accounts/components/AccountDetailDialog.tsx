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
import { PasswordInput } from '@/components/shared/PasswordInput'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { CodexQuota } from './CodexQuota'
import { VoyageQuota } from './VoyageQuota'
import { bucketOf } from '@/config/providers'
import { useAccountFull } from '@/hooks/queries/useAccounts'
import {
  useUpdateAccount,
  useSetAccountOverage,
  useConsumeCodexResetCredit,
} from '@/hooks/mutations/useAccountMutations'
import { generateMachineId } from '@/services/accounts.service'

const PRIORITY_MODELS = ['claude-sonnet-4-5', 'claude-opus-4.8', 'claude-haiku-4-5']

interface Props {
  account: AccountListItem | null
  onClose: () => void
}

export function AccountDetailDialog({ account, onClose }: Props) {
  const { t } = useTranslation()
  const full = useAccountFull(account?.id ?? null)
  const update = useUpdateAccount()
  const overage = useSetAccountOverage()
  const consumeResetCredit = useConsumeCodexResetCredit()

  const [nickname, setNickname] = useState('')
  const [weight, setWeight] = useState(1)
  const [machineId, setMachineId] = useState('')
  const [proxyURL, setProxyURL] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [confirmResetOpen, setConfirmResetOpen] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [checkKeyURL, setCheckKeyURL] = useState('')
  const [customModelsText, setCustomModelsText] = useState('')

  useEffect(() => {
    if (account) {
      setNickname(account.nickname || '')
      setWeight(account.weight || 1)
      setMachineId(account.machineId || '')
      setProxyURL(account.proxyURL || '')
      setBaseURL(account.remoteBaseURL || '')
      setCheckKeyURL(account.remoteCheckKeyURL || '')
      setCustomModelsText((account.customModels || []).join(', '))
    }
  }, [account])

  useEffect(() => {
    if (full.data) {
      if (full.data.accessToken) setApiKey(full.data.accessToken)
      if (full.data.remoteBaseURL) setBaseURL(full.data.remoteBaseURL)
      if (full.data.remoteCheckKeyURL) setCheckKeyURL(full.data.remoteCheckKeyURL)
      if (full.data.customModels) setCustomModelsText(full.data.customModels.join(', '))
    }
  }, [full.data])

  const detailAccount = full.data ?? account
  const isRemoteKiro = detailAccount
    ? bucketOf(detailAccount.provider) === 'remotekiro' || detailAccount.authMethod?.toLowerCase() === 'remotekiro'
    : false

  function togglePresetModel(model: string) {
    const list = customModelsText
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    if (list.includes(model)) {
      setCustomModelsText(list.filter((m) => m !== model).join(', '))
    } else {
      setCustomModelsText([...list, model].join(', '))
    }
  }

  const currentModels = customModelsText
    .split(/[\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean)

  function save() {
    if (!account) return
    const customModels = customModelsText
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    update.mutate(
      {
        id: account.id,
        patch: {
          nickname,
          weight,
          machineId,
          proxyURL,
          customModels,
          remoteBaseURL: isRemoteKiro ? baseURL.trim() : undefined,
          accessToken: isRemoteKiro ? apiKey.trim() : undefined,
          remoteCheckKeyURL: isRemoteKiro ? checkKeyURL.trim() : undefined,
        },
      },
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

  const overageOn = (detailAccount?.overageStatus ?? account?.overageStatus) === 'enabled'
  const overageCapable = (detailAccount?.overageCapability ?? account?.overageCapability) !== 'none'
  const isCodex = detailAccount
    ? bucketOf(detailAccount.provider) === 'codex' || detailAccount.authMethod?.toLowerCase() === 'codex'
    : false
  const isVoyage = detailAccount
    ? bucketOf(detailAccount.provider) === 'voyage' || detailAccount.authMethod?.toLowerCase() === 'voyage'
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
                onResetQuota={() => setConfirmResetOpen(true)}
                resetting={consumeResetCredit.isPending}
              />
            )}

            {isVoyage && detailAccount && (
              <VoyageQuota buckets={detailAccount.voyageQuota} detail />
            )}

            <div className="space-y-2">
              <Label htmlFor="nickname">{t('detail.nickname')}</Label>
              <Input id="nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} />
            </div>

            {isRemoteKiro && (
              <>
                <div className="space-y-2">
                  <Label htmlFor="baseURL">Base URL</Label>
                  <Input
                    id="baseURL"
                    value={baseURL}
                    onChange={(e) => setBaseURL(e.target.value)}
                    placeholder="https://..."
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="apiKey">API Key</Label>
                  <PasswordInput
                    id="apiKey"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder="sk-..."
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="checkKeyURL">Check Key URL</Label>
                  <Input
                    id="checkKeyURL"
                    value={checkKeyURL}
                    onChange={(e) => setCheckKeyURL(e.target.value)}
                    placeholder="(optional)"
                  />
                </div>
              </>
            )}

            <div className="space-y-2">
              <Label htmlFor="customModels">{t('remotekiro.modelsLabel')}</Label>
              <Input
                id="customModels"
                value={customModelsText}
                onChange={(e) => setCustomModelsText(e.target.value)}
                placeholder={t('remotekiro.modelsPlaceholder')}
              />
              <div className="flex flex-wrap items-center gap-1.5 pt-1">
                <span className="text-xs text-muted-foreground">{t('remotekiro.priorityModels')}</span>
                {PRIORITY_MODELS.map((pm) => {
                  const active = currentModels.includes(pm)
                  return (
                    <button
                      type="button"
                      key={pm}
                      onClick={() => togglePresetModel(pm)}
                      className={`text-xs px-2 py-0.5 rounded border transition-colors cursor-pointer ${
                        active
                          ? 'bg-primary/15 border-primary text-primary font-medium'
                          : 'bg-muted/40 border-border text-muted-foreground hover:bg-muted'
                      }`}
                    >
                      {active ? '✓ ' : '+ '}
                      {pm}
                    </button>
                  )
                })}
              </div>
              <p className="text-xs text-muted-foreground">{t('remotekiro.modelsHint')}</p>
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

        <ConfirmDialog
          open={confirmResetOpen}
          onOpenChange={setConfirmResetOpen}
          title={t('codex.confirmResetTitle')}
          description={t('codex.confirmResetDesc')}
          confirmLabel={t('codex.consumeReset')}
          onConfirm={() => {
            if (!detailAccount) return
            consumeResetCredit.mutate(
              { id: detailAccount.id },
              {
                onSuccess: () => {
                  toast.success(t('codex.resetSuccess'))
                  setConfirmResetOpen(false)
                },
                onError: (err: any) => {
                  toast.error(err?.message || t('codex.resetFailed'))
                  setConfirmResetOpen(false)
                },
              }
            )
          }}
        />
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
