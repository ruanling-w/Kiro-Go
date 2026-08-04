// ConnectionCard — base URL + API key reveal + model picker for snippet fill.
// Base URL is the public *gateway* origin (not the Vite admin origin in dev).
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useUiStore } from '@/stores/uiStore'
import { useApiKeys } from '@/hooks/queries/useApiKeys'
import { useSettings } from '@/hooks/queries/useSettings'
import { usePublicModels } from '@/hooks/queries/usePublicModels'
import { revealApiKey } from '@/services/apikeys.service'
import {
  isDevAdminOrigin,
  isLocalGatewayBase,
  resolveGatewayBaseURL,
} from '@/lib/gatewayBase'
import type { PublicModel } from '@/types/publicApi'

export interface ConnectionValues {
  base: string
  key: string
  model: string
  models: PublicModel[]
  modelsLoading: boolean
  modelsError: boolean
}

interface ConnectionCardProps {
  onChange: (v: ConnectionValues) => void
}

export function ConnectionCard({ onChange }: ConnectionCardProps) {
  const { t } = useTranslation()
  const docsBaseURL = useUiStore((s) => s.docsBaseURL)
  const setDocsBaseURL = useUiStore((s) => s.setDocsBaseURL)
  const docsApiKeyId = useUiStore((s) => s.docsApiKeyId)
  const setDocsApiKeyId = useUiStore((s) => s.setDocsApiKeyId)

  const apiKeys = useApiKeys()
  const settings = useSettings()

  const gatewayBase = useMemo(
    () => resolveGatewayBaseURL({ port: settings.data?.port }),
    [settings.data?.port],
  )

  // Auto-fill / correct base: empty, or stale Vite admin origin saved in localStorage.
  useEffect(() => {
    if (!docsBaseURL || isDevAdminOrigin(docsBaseURL)) {
      setDocsBaseURL(gatewayBase)
    }
  }, [docsBaseURL, gatewayBase, setDocsBaseURL])

  // Snippets use the gateway base; model catalog in Vite dev goes same-origin
  // through the /v1 proxy to avoid CORS against :8080.
  const modelsFetchBase = useMemo(() => {
    if (import.meta.env.DEV && isLocalGatewayBase(docsBaseURL || gatewayBase)) return ''
    return docsBaseURL || gatewayBase
  }, [docsBaseURL, gatewayBase])

  const modelsQ = usePublicModels(modelsFetchBase)

  const [revealedKey, setRevealedKey] = useState<string | null>(null)
  const [revealing, setRevealing] = useState(false)
  const [modelId, setModelId] = useState('')

  const enabledKeys = useMemo(
    () => (apiKeys.data ?? []).filter((k) => k.enabled && !k.expired),
    [apiKeys.data],
  )

  const models = useMemo(() => modelsQ.data?.data ?? [], [modelsQ.data])

  const effectiveBase = docsBaseURL || gatewayBase

  // Pick a sensible default model once the catalog loads.
  useEffect(() => {
    if (!models.length) return
    if (modelId && models.some((m) => m.id === modelId)) return
    const preferred =
      models.find((m) => m.id === 'claude-sonnet-4.5') ??
      models.find((m) => !m.combo && !m.id.includes('-thinking')) ??
      models[0]
    if (preferred) setModelId(preferred.id)
  }, [models, modelId])

  // Clear revealed key when selection changes.
  useEffect(() => {
    setRevealedKey(null)
  }, [docsApiKeyId])

  useEffect(() => {
    onChange({
      base: effectiveBase,
      key: revealedKey ?? '',
      model: modelId,
      models,
      modelsLoading: modelsQ.isPending,
      modelsError: modelsQ.isError,
    })
  }, [
    effectiveBase,
    revealedKey,
    modelId,
    models,
    modelsQ.isPending,
    modelsQ.isError,
    onChange,
  ])

  async function handleReveal() {
    if (!docsApiKeyId) return
    if (revealedKey) {
      setRevealedKey(null)
      return
    }
    setRevealing(true)
    try {
      const res = await revealApiKey(docsApiKeyId)
      setRevealedKey(res.key)
    } catch {
      toast.error(t('apiKeys.revealFailed'))
    } finally {
      setRevealing(false)
    }
  }

  const requireApiKey = settings.data?.requireApiKey ?? true

  return (
    <Card className="min-w-0">
      <CardHeader className="gap-1.5">
        <CardTitle>{t('apiDocs.connectionTitle')}</CardTitle>
        <CardDescription>{t('apiDocs.connectionDesc')}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0 space-y-4">
        {!requireApiKey && (
          <div
            role="status"
            className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm leading-snug text-amber-800 dark:text-amber-300"
          >
            {t('apiDocs.requireApiKeyOff')}
          </div>
        )}

        <div className="grid min-w-0 gap-4 sm:grid-cols-2">
          <div className="min-w-0 space-y-1.5 sm:col-span-2">
            <Label htmlFor="docs-base-url">{t('apiDocs.baseUrl')}</Label>
            <div className="flex min-w-0 items-center gap-2">
              <Input
                id="docs-base-url"
                value={docsBaseURL}
                onChange={(e) => setDocsBaseURL(e.target.value)}
                placeholder={gatewayBase || 'http://localhost:8080'}
                className="min-w-0 flex-1 font-mono text-sm"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="shrink-0"
                onClick={() => setDocsBaseURL(gatewayBase)}
                aria-label={t('apiDocs.resetBase')}
                title={t('apiDocs.resetBase')}
              >
                <RotateCcw className="size-4" />
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">{t('apiDocs.baseUrlHint')}</p>
          </div>

          <div className="min-w-0 space-y-1.5">
            <Label>{t('apiDocs.apiKey')}</Label>
            <div className="flex min-w-0 items-center gap-2">
              <Select
                value={docsApiKeyId || 'none'}
                onValueChange={(v) => setDocsApiKeyId(v === 'none' ? null : v)}
              >
                <SelectTrigger className="min-w-0 flex-1">
                  <SelectValue placeholder={t('apiDocs.selectKey')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">{t('apiDocs.selectKey')}</SelectItem>
                  {enabledKeys.map((k) => (
                    <SelectItem key={k.id} value={k.id}>
                      <span className="truncate">
                        {k.name || k.keyMasked} · {k.keyMasked}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="shrink-0"
                disabled={!docsApiKeyId || revealing}
                onClick={() => void handleReveal()}
                aria-label={revealedKey ? t('apiKeys.hideKey') : t('apiKeys.showKey')}
              >
                {revealedKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
              </Button>
            </div>
            <p className="break-all font-mono text-xs text-muted-foreground">
              {revealedKey
                ? revealedKey
                : docsApiKeyId
                  ? t('apiDocs.keyHidden')
                  : t('apiDocs.keyPlaceholder')}
            </p>
            {enabledKeys.length === 0 ? (
              <p className="text-xs text-muted-foreground">{t('apiDocs.noKeys')}</p>
            ) : null}
          </div>

          <div className="min-w-0 space-y-1.5">
            <Label>{t('apiDocs.model')}</Label>
            <Select value={modelId} onValueChange={setModelId} disabled={!models.length}>
              <SelectTrigger className="w-full min-w-0">
                <SelectValue
                  placeholder={
                    modelsQ.isPending
                      ? t('api.loading')
                      : modelsQ.isError
                        ? t('api.fetchError')
                        : t('apiDocs.selectModel')
                  }
                />
              </SelectTrigger>
              <SelectContent className="max-h-72">
                {models.map((m) => (
                  <SelectItem key={m.id} value={m.id}>
                    <span className="truncate">
                      {m.id}
                      {m.combo ? ' · combo' : ''}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
