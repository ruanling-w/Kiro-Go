// TestAccountDialog — pick a model and run a live test against one account. The
// test hits POST /accounts/{id}/test; a small log area shows start → success /
// failed. Models come from the account's cached model list (useAccountModels).
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, XCircle } from 'lucide-react'
import type { AccountListItem } from '@/types/account'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import { useAccountModels } from '@/hooks/queries/useProviderModels'
import { testAccount, type TestAccountResult } from '@/services/accounts.service'

interface Props {
  account: AccountListItem | null
  onClose: () => void
}

interface LogLine {
  tone: 'info' | 'success' | 'error'
  text: string
}

export function TestAccountDialog({ account, onClose }: Props) {
  const { t } = useTranslation()
  const models = useAccountModels(account?.id ?? '', !!account)
  const [model, setModel] = useState('')
  const [running, setRunning] = useState(false)
  const [log, setLog] = useState<LogLine[]>([])

  useEffect(() => {
    setLog([])
    setModel('')
  }, [account?.id])

  const modelList = models.data ?? []

  async function runTest() {
    if (!account) return
    setRunning(true)
    setLog([{ tone: 'info', text: t('accounts.testLog.start') }])
    try {
      const res: TestAccountResult = await testAccount(account.id, model || undefined)
      if (res.success) {
        setLog((l) => [
          ...l,
          { tone: 'success', text: `${t('accounts.testLog.success')}${res.model ? ` · ${res.model}` : ''}${res.latency ? ` · ${res.latency}ms` : ''}` },
        ])
      } else {
        setLog((l) => [...l, { tone: 'error', text: res.error || res.message || t('accounts.testLog.failed') }])
      }
    } catch (err) {
      setLog((l) => [...l, { tone: 'error', text: err instanceof Error ? err.message : t('accounts.testLog.error') }])
    } finally {
      setRunning(false)
    }
  }

  return (
    <Dialog open={!!account} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('accounts.testModalTitle')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label>{t('accounts.selectModel')}</Label>
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger>
                <SelectValue
                  placeholder={
                    models.isPending ? t('accounts.testModelsLoading') : t('accounts.selectModel')
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {modelList.map((m) => (
                  <SelectItem key={m.modelId} value={m.modelId}>
                    {m.modelName || m.modelId}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {log.length > 0 && (
            <div className="max-h-52 space-y-1 overflow-auto rounded-lg border bg-muted/40 p-3 font-mono text-xs">
              {log.map((line, i) => (
                <div
                  key={i}
                  className={
                    line.tone === 'success'
                      ? 'flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400'
                      : line.tone === 'error'
                        ? 'flex items-center gap-1.5 text-destructive'
                        : 'flex items-center gap-1.5 text-muted-foreground'
                  }
                >
                  {line.tone === 'success' && <CheckCircle2 className="size-3.5" />}
                  {line.tone === 'error' && <XCircle className="size-3.5" />}
                  <span>{line.text}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('common.close')}
          </Button>
          <Button onClick={runTest} disabled={running}>
            {running ? (
              <span className="flex items-center gap-2">
                <HamsterWheel size="sm" />
                {t('accounts.testing')}
              </span>
            ) : (
              t('accounts.test')
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
