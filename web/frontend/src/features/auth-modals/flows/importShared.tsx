// Shared pieces for the direct-import flows (no OAuth session, just form →
// mutation → done). Extracted from ImportFlows.tsx so the flows that live in
// their own files (LocalCacheFlow, WebCookieFlow) reuse the same terminal states
// instead of re-implementing them.
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, XCircle } from 'lucide-react'
import { qk } from '@/config/queryKeys'
import { ApiError } from '@/services/httpClient'
import { Button } from '@/components/ui/button'

/** Wraps an import request as a mutation that refreshes the accounts list. */
export function useImport<T>(fn: (body: T) => Promise<{ success: boolean }>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

export function Done({ onDone, label }: { onDone?: () => void; label: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <CheckCircle2 className="size-12 text-emerald-500" />
      <p className="font-medium">{label}</p>
      <Button onClick={onDone}>{t('common.close')}</Button>
    </div>
  )
}

export function ErrorNote({ error }: { error: unknown }) {
  const { t } = useTranslation()
  if (!error) return null
  const msg = error instanceof ApiError ? error.message : t('common.failed')
  return (
    <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
      <XCircle className="mt-0.5 size-4 shrink-0" />
      <span>{msg}</span>
    </div>
  )
}
