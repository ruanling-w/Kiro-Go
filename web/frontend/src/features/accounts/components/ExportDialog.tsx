// ExportDialog — select accounts (or all) and export their credentials as JSON:
// show / copy / download. POST /export returns the full export object; empty ids
// = export all. Timestamps in the export are MILLISECONDS (see plan/format).
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Download, Copy, Check } from 'lucide-react'
import type { AccountListItem } from '@/types/account'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import { exportAccounts } from '@/services/export.service'
import { maskEmail } from '@/lib/mask'
import { useUiStore } from '@/stores/uiStore'
import { toast } from 'sonner'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  accounts: AccountListItem[]
}

export function ExportDialog({ open, onOpenChange, accounts }: Props) {
  const { t } = useTranslation()
  const privacy = useUiStore((s) => s.privacyMode)
  const { copied, copy } = useCopyToClipboard()
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [json, setJson] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (open) {
      setSelected(new Set())
      setJson('')
    }
  }, [open])

  const allSelected = accounts.length > 0 && selected.size === accounts.length

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function runExport() {
    setLoading(true)
    try {
      const ids = selected.size > 0 ? [...selected] : []
      const data = await exportAccounts(ids)
      setJson(JSON.stringify(data, null, 2))
    } catch {
      toast.error(t('common.failed'))
    } finally {
      setLoading(false)
    }
  }

  function download() {
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `kiro-accounts-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* sm:max-w-* is required, not just max-w-*: DialogContent's own default
          sm:max-w-sm wins over an unprefixed max-w-lg from 640px up. Wide because
          the JSON lines (access/refresh tokens) are very long. */}
      <DialogContent className="max-w-[calc(100%-2rem)] sm:max-w-3xl lg:max-w-5xl">
        <DialogHeader>
          <DialogTitle>{t('export.title')}</DialogTitle>
        </DialogHeader>

        {json ? (
          <pre className="max-h-[60vh] overflow-auto rounded-lg border bg-muted/40 p-3 font-mono text-xs">
            {json}
          </pre>
        ) : (
          <div className="space-y-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                setSelected(allSelected ? new Set() : new Set(accounts.map((a) => a.id)))
              }
            >
              {allSelected ? t('export.deselectAll') : t('export.selectAll')}
            </Button>
            <div className="max-h-64 space-y-1 overflow-auto rounded-lg border p-2">
              {accounts.map((a) => (
                <label
                  key={a.id}
                  className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-muted"
                >
                  <input
                    type="checkbox"
                    checked={selected.has(a.id)}
                    onChange={() => toggle(a.id)}
                  />
                  <span className="truncate">
                    {a.nickname || (privacy ? maskEmail(a.email || a.id) : a.email || a.id)}
                  </span>
                </label>
              ))}
            </div>
          </div>
        )}

        <DialogFooter>
          {json ? (
            <>
              <Button variant="outline" onClick={() => void copy(json)}>
                {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                {t('export.copyJson')}
              </Button>
              <Button onClick={download}>
                <Download className="size-4" />
                {t('export.downloadJson')}
              </Button>
            </>
          ) : (
            <Button onClick={runExport} disabled={loading}>
              {loading ? <HamsterLoader size="sm" /> : t('export.showJson')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
