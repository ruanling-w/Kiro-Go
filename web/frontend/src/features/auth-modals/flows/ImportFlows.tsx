// Direct import flows — no OAuth session, just a form → mutation → done. Each
// posts to its import endpoint and invalidates the accounts list on success.
// Shared terminal states (Done/ErrorNote/useImport) live in importShared.tsx so
// the standalone flows (LocalCacheFlow, WebCookieFlow) reuse them.
import { useState, type ChangeEvent, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Upload } from 'lucide-react'
import { qk } from '@/config/queryKeys'
import {
  importKiroApiKey,
  importRemoteKiro,
  importSsoToken,
  importAccountsJson,
  importCodex,
  type CodexImport,
} from '@/services/authFlows.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { RegionSelect } from '@/components/shared/RegionSelect'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import type { FlowComponentProps } from './types'
import { HelpBlock, Steps, Code } from './HelpBlock'
import { Done, ErrorNote, useImport } from './importShared'
import { parseCredentialsInput, normalizeForPost } from './parseCredentials'
import { tp } from '@/lib/t'

// --- Kiro API key import ---
export function KiroApiKeyFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importKiroApiKey)
  const [apiKey, setApiKey] = useState('')
  const [region, setRegion] = useState('us-east-1')
  const [nickname, setNickname] = useState('')

  if (m.isSuccess) return <Done onDone={onDone} label={t('accounts.testSuccess')} />

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({ apiKey: apiKey.trim(), region, nickname: nickname.trim() || undefined })
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label>API Key</Label>
        <PasswordInput value={apiKey} onChange={(e) => setApiKey(e.target.value)} autoFocus />
      </div>
      <div className="space-y-2">
        <Label>{t('detail.region')}</Label>
        <RegionSelect value={region} onChange={setRegion} />
      </div>
      <div className="space-y-2">
        <Label>{t('detail.nickname')}</Label>
        <Input value={nickname} onChange={(e) => setNickname(e.target.value)} />
      </div>
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!apiKey.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

// --- Remote Kiro import ---
export function RemoteKiroFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importRemoteKiro)
  const [baseURL, setBaseURL] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [nickname, setNickname] = useState('')
  const [checkKeyURL, setCheckKeyURL] = useState('')

  if (m.isSuccess) return <Done onDone={onDone} label={t('accounts.testSuccess')} />

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({
      baseURL: baseURL.trim(),
      apiKey: apiKey.trim(),
      nickname: nickname.trim() || undefined,
      checkKeyURL: checkKeyURL.trim() || undefined,
    })
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label>Base URL</Label>
        <Input value={baseURL} onChange={(e) => setBaseURL(e.target.value)} placeholder="https://..." autoFocus />
      </div>
      <div className="space-y-2">
        <Label>API Key</Label>
        <PasswordInput value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>{t('detail.nickname')}</Label>
        <Input value={nickname} onChange={(e) => setNickname(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>Check Key URL</Label>
        <Input value={checkKeyURL} onChange={(e) => setCheckKeyURL(e.target.value)} placeholder="(optional)" />
      </div>
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!baseURL.trim() || !apiKey.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

// --- SSO token import (batch: multiple accounts + per-line errors) ---
export function SsoTokenFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [bearerToken, setBearerToken] = useState('')
  const [region, setRegion] = useState('us-east-1')
  const m = useMutation({
    mutationFn: importSsoToken,
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })

  if (m.isSuccess) {
    const count = m.data.accounts.length
    const errs = m.data.errors?.length ?? 0
    return (
      <div className="space-y-3 py-4">
        <div className="flex flex-col items-center gap-2 text-center">
          <CheckCircle2 className="size-10 text-emerald-500" />
          <p className="font-medium">
            {tp(t, 'sso.importSuccess', count)}
            {errs > 0 && tp(t, 'sso.importPartial', errs)}
          </p>
        </div>
        {m.data.errors && m.data.errors.length > 0 && (
          <ul className="space-y-1 text-xs text-destructive">
            {m.data.errors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        )}
        <Button className="w-full" onClick={onDone}>{t('common.close')}</Button>
      </div>
    )
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({ bearerToken: bearerToken.trim(), region })
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      <HelpBlock title={t('sso.howToGet')}>
        <Steps>
          <li>
            {t('sso.step1')} <Code>&lt;tenant&gt;.awsapps.com/start</Code>
          </li>
          <li>{t('sso.step2')}</li>
          <li>{t('sso.step3')}</li>
        </Steps>
      </HelpBlock>
      <div className="space-y-2">
        <Label>{t('sso.tokenLabel')}</Label>
        <textarea
          value={bearerToken}
          onChange={(e) => setBearerToken(e.target.value)}
          placeholder={t('sso.tokenPlaceholder')}
          className="min-h-24 w-full rounded-lg border bg-transparent px-3 py-2 font-mono text-xs"
          autoFocus
        />
        <p className="text-xs text-muted-foreground">{t('sso.tokenHint')}</p>
      </div>
      <div className="space-y-2">
        <Label>{t('detail.region')}</Label>
        <RegionSelect value={region} onChange={setRegion} />
      </div>
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!bearerToken.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

// --- Raw credentials import (JSON bundle / array / single object / ---- lines) ---
// This is also the JSON *import* counterpart to /export: paste or upload the
// exported bundle and every account in it is submitted in one batch. refreshToken
// is the only required field — the server refreshes to mint an accessToken, which
// is why the legacy importer deliberately sent an empty one.
export function CredentialsFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [raw, setRaw] = useState('')
  const [region, setRegion] = useState('us-east-1')
  const [localError, setLocalError] = useState('')
  const [note, setNote] = useState('')
  const m = useMutation({
    mutationFn: importAccountsJson,
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })

  if (m.isSuccess) {
    const count = m.data.accounts.length
    const errs = m.data.errors?.length ?? 0
    return (
      <div className="space-y-3 py-4">
        <div className="flex flex-col items-center gap-2 text-center">
          <CheckCircle2 className="size-10 text-emerald-500" />
          <p className="font-medium">
            {tp(t, 'sso.importSuccess', count)}
            {errs > 0 && tp(t, 'sso.importPartial', errs)}
            {note}
          </p>
        </div>
        {m.data.errors && m.data.errors.length > 0 && (
          <ul className="max-h-40 space-y-1 overflow-y-auto text-xs text-destructive">
            {m.data.errors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        )}
        <Button className="w-full" onClick={onDone}>{t('common.close')}</Button>
      </div>
    )
  }

  function loadFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    void file.text().then(setRaw)
    e.target.value = '' // allow re-picking the same file
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    setLocalError('')
    setNote('')

    let parsed
    try {
      parsed = parseCredentialsInput(raw)
    } catch {
      return setLocalError(t('credentials.jsonError'))
    }
    if (parsed.items.length === 0) {
      return setLocalError(
        parsed.usedLineFormat
          ? tp(t, 'credentials.lineParseAllSkipped', parsed.skipped)
          : t('credentials.jsonError'),
      )
    }
    if (parsed.skipped > 0) setNote(tp(t, 'credentials.lineParseSkipped', parsed.skipped))

    m.mutate({
      accounts: parsed.items.map((item) => {
        const body = normalizeForPost(item)
        // Only fall back to the picked region when the item carried none.
        return { ...body, region: item.region || region }
      }),
    })
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-xs text-muted-foreground">{t('credentials.batchHint')}</p>

      <div className="space-y-2">
        <Label>{t('credentials.label')}</Label>
        <textarea
          value={raw}
          onChange={(e) => setRaw(e.target.value)}
          placeholder={'[{"refreshToken":"xxx","provider":"BuilderID"}]\nemail----password----refreshToken----clientId----clientSecret'}
          autoFocus
          className="min-h-32 w-full rounded-lg border bg-transparent px-3 py-2 font-mono text-xs"
        />
        <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs hover:bg-muted">
          <Upload className="size-3.5" />
          {t('local.upload')}
          <input type="file" accept=".json,application/json,.txt" onChange={loadFile} className="hidden" />
        </label>
      </div>

      <div className="space-y-2">
        <Label>{t('detail.region')}</Label>
        <RegionSelect value={region} onChange={setRegion} />
      </div>

      {localError && (
        <p className="text-sm text-destructive" role="alert">
          {localError}
        </p>
      )}
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!raw.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

// --- Codex credentials import ---
export function CodexImportFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const m = useMutation({
    mutationFn: async (payloads: CodexImport[]) => {
      let ok = 0
      let fail = 0
      for (const payload of payloads) {
        try {
          const res = await importCodex(payload)
          if (res?.success) ok++
          else fail++
        } catch {
          fail++
        }
      }
      if (ok === 0) throw new Error('No valid Codex account imported')
      return { ok, fail }
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
  const [rawJson, setRawJson] = useState('')
  const [accessToken, setAccessToken] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [idToken, setIdToken] = useState('')
  const [localError, setLocalError] = useState('')

  if (m.isSuccess) {
    const ok = m.data?.ok || 0
    const fail = m.data?.fail || 0
    const label = fail > 0 ? `Imported ${ok} account(s), failed ${fail}.` : `Imported ${ok} account(s).`
    return <Done onDone={onDone} label={label} />
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    setLocalError('')

    let payloads: CodexImport[] = [
      {
        accessToken: accessToken.trim(),
        refreshToken: refreshToken.trim(),
        idToken: idToken.trim() || undefined,
      },
    ]

    if (rawJson.trim()) {
      try {
        const parsed = JSON.parse(rawJson.trim())
        if (!parsed || typeof parsed !== 'object') {
          throw new Error('invalid')
        }
        const items = Array.isArray(parsed) ? parsed : [parsed]
        payloads = items
          .map((item) => {
            const rawExpires = (item as Record<string, unknown>)?.expiresIn ?? (item as Record<string, unknown>)?.expires_in
            const expiresNum = typeof rawExpires === 'number' ? rawExpires : Number(rawExpires)
            return {
              accessToken: String((item as Record<string, unknown>)?.accessToken ?? (item as Record<string, unknown>)?.access_token ?? '').trim(),
              refreshToken: String((item as Record<string, unknown>)?.refreshToken ?? (item as Record<string, unknown>)?.refresh_token ?? '').trim(),
              idToken: String((item as Record<string, unknown>)?.idToken ?? (item as Record<string, unknown>)?.id_token ?? '').trim() || undefined,
              expiresIn: Number.isFinite(expiresNum) && expiresNum > 0 ? expiresNum : undefined,
            } satisfies CodexImport
          })
          .filter((item) => item.accessToken)
      } catch {
        return setLocalError('Invalid JSON. Paste a JSON object or array with accessToken/refreshToken/idToken fields.')
      }
    }

    if (payloads.length === 0) {
      return setLocalError('accessToken is required')
    }

    m.mutate(payloads)
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label>JSON credentials</Label>
        <textarea
          value={rawJson}
          onChange={(e) => setRawJson(e.target.value)}
          placeholder={`[{"accessToken":"...","refreshToken":"...","idToken":"...","expiresIn":3600}]`}
          className="min-h-24 w-full rounded-lg border bg-transparent px-3 py-2 font-mono text-xs"
          autoFocus
        />
        <p className="text-xs text-muted-foreground">Paste a JSON object or array with accessToken / refreshToken / idToken. You can also fill the fields below manually.</p>
      </div>
      <div className="space-y-2">
        <Label>Access Token</Label>
        <PasswordInput value={accessToken} onChange={(e) => setAccessToken(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>Refresh Token</Label>
        <PasswordInput value={refreshToken} onChange={(e) => setRefreshToken(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>ID Token</Label>
        <PasswordInput value={idToken} onChange={(e) => setIdToken(e.target.value)} />
      </div>
      {localError && (
        <p className="text-sm text-destructive" role="alert">
          {localError}
        </p>
      )}
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={(!accessToken.trim() && !rawJson.trim()) || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}
