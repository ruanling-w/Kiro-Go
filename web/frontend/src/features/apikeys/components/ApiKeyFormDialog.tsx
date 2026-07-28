// ApiKeyFormDialog — create/edit an API key. On create, the backend returns the
// cleartext key ONCE; we surface it in a follow-up "show once" panel with copy.
// Edit pre-fills from the row; omitted fields leave values unchanged.
//
// Limits use 0 = unlimited (matching the backend); the expiry input is a
// datetime-local converted to/from unix seconds (0 = never).
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy, TriangleAlert } from 'lucide-react'
import type { ApiKeyView } from '@/types/apikey'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { useCreateApiKey, useUpdateApiKey } from '@/hooks/mutations/useApiKeyMutations'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { ApiError } from '@/services/httpClient'
import { toast } from 'sonner'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  editing: ApiKeyView | null
}

function toDatetimeLocal(unixSeconds: number): string {
  if (!unixSeconds) return ''
  const d = new Date(unixSeconds * 1000)
  const off = d.getTimezoneOffset()
  return new Date(d.getTime() - off * 60_000).toISOString().slice(0, 16)
}

function fromDatetimeLocal(value: string): number {
  if (!value) return 0
  return Math.floor(new Date(value).getTime() / 1000)
}

export function ApiKeyFormDialog({ open, onOpenChange, editing }: Props) {
  const { t } = useTranslation()
  const create = useCreateApiKey()
  const update = useUpdateApiKey()
  const { copied, copy } = useCopyToClipboard()

  const [name, setName] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [tokenLimit, setTokenLimit] = useState('0')
  const [creditLimit, setCreditLimit] = useState('0')
  const [expiry, setExpiry] = useState('')
  const [createdKey, setCreatedKey] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setCreatedKey(null)
    if (editing) {
      setName(editing.name)
      setEnabled(editing.enabled)
      setTokenLimit(String(editing.tokenLimit ?? 0))
      setCreditLimit(String(editing.creditLimit ?? 0))
      setExpiry(toDatetimeLocal(editing.expiresAt))
    } else {
      setName('')
      setEnabled(true)
      setTokenLimit('0')
      setCreditLimit('0')
      setExpiry('')
    }
  }, [open, editing])

  function handleSubmit() {
    const body = {
      name: name.trim(),
      enabled,
      tokenLimit: Number(tokenLimit) || 0,
      creditLimit: Number(creditLimit) || 0,
      expiresAt: fromDatetimeLocal(expiry),
    }
    if (editing) {
      update.mutate(
        { id: editing.id, patch: body },
        {
          onSuccess: () => {
            toast.success(t('apiKeys.updated'))
            onOpenChange(false)
          },
          onError: (e) => toast.error(e instanceof ApiError ? e.message : t('common.failed')),
        },
      )
    } else {
      create.mutate(body, {
        onSuccess: (res) => {
          setCreatedKey(res.key)
          toast.success(t('apiKeys.created'))
        },
        onError: (e) => toast.error(e instanceof ApiError ? e.message : t('common.failed')),
      })
    }
  }

  const busy = create.isPending || update.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {editing ? t('apiKeys.modalTitleEdit') : t('apiKeys.modalTitleCreate')}
          </DialogTitle>
        </DialogHeader>

        {createdKey ? (
          <div className="space-y-3">
            <div className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <span>{t('apiKeys.showWarning')}</span>
            </div>
            <div className="flex items-center gap-2 rounded-lg border bg-muted/50 p-3">
              <code className="min-w-0 flex-1 break-all font-mono text-sm">{createdKey}</code>
              <Button
                variant="outline"
                size="icon-sm"
                onClick={() => void copy(createdKey)}
                aria-label={t('apiKeys.copyBtn')}
              >
                {copied ? <Check className="size-4 text-emerald-500" /> : <Copy className="size-4" />}
              </Button>
            </div>
            <DialogFooter>
              <Button onClick={() => onOpenChange(false)}>{t('apiKeys.closeBtn')}</Button>
            </DialogFooter>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="ak-name">{t('apiKeys.formName')}</Label>
              <Input
                id="ak-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('apiKeys.formNamePlaceholder')}
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="ak-tok">{t('apiKeys.limitTokens')}</Label>
                <Input
                  id="ak-tok"
                  type="number"
                  min={0}
                  value={tokenLimit}
                  onChange={(e) => setTokenLimit(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ak-cred">{t('apiKeys.limitCredits')}</Label>
                <Input
                  id="ak-cred"
                  type="number"
                  min={0}
                  value={creditLimit}
                  onChange={(e) => setCreditLimit(e.target.value)}
                />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t('apiKeys.limitHint')}</p>
            <div className="space-y-2">
              <Label htmlFor="ak-exp">{t('apiKeys.formExpiry')}</Label>
              <Input
                id="ak-exp"
                type="datetime-local"
                value={expiry}
                onChange={(e) => setExpiry(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">{t('apiKeys.expiryHint')}</p>
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="ak-en">{t('apiKeys.formEnabled')}</Label>
              <Switch id="ak-en" checked={enabled} onCheckedChange={setEnabled} />
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                {t('apiKeys.cancelBtn')}
              </Button>
              <Button onClick={handleSubmit} disabled={busy}>
                {t('apiKeys.saveBtn')}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
