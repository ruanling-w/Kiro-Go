// Direct import flows — no OAuth session, just a form → mutation → done. Each
// posts to its import endpoint and invalidates the accounts list on success.
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, XCircle } from 'lucide-react'
import { qk } from '@/config/queryKeys'
import { ApiError } from '@/services/httpClient'
import {
  importKiroApiKey,
  importRemoteKiro,
  importSsoToken,
  importCredentials,
  importCodex,
} from '@/services/authFlows.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { RegionSelect } from '@/components/shared/RegionSelect'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import type { FlowComponentProps } from './types'

// Small helper wrapping a form + submit + terminal states, shared by all imports.
function useImport<T>(fn: (body: T) => Promise<{ success: boolean }>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

function Done({ onDone, label }: { onDone?: () => void; label: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center gap-3 py-8 text-center">
      <CheckCircle2 className="size-12 text-emerald-500" />
      <p className="font-medium">{label}</p>
      <Button onClick={onDone}>{t('common.close')}</Button>
    </div>
  )
}

function ErrorNote({ error }: { error: unknown }) {
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
    return (
      <div className="space-y-3 py-4">
        <div className="flex flex-col items-center gap-2 text-center">
          <CheckCircle2 className="size-10 text-emerald-500" />
          <p className="font-medium">{m.data.accounts.length} ✓</p>
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
      <div className="space-y-2">
        <Label>Bearer Token</Label>
        <textarea
          value={bearerToken}
          onChange={(e) => setBearerToken(e.target.value)}
          className="min-h-24 w-full rounded-lg border bg-transparent px-3 py-2 text-sm"
          autoFocus
        />
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

// --- Raw credentials import ---
export function CredentialsFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importCredentials)
  const [accessToken, setAccessToken] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [region, setRegion] = useState('us-east-1')

  if (m.isSuccess) return <Done onDone={onDone} label={t('accounts.testSuccess')} />

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({ accessToken: accessToken.trim(), refreshToken: refreshToken.trim(), region })
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label>Access Token</Label>
        <PasswordInput value={accessToken} onChange={(e) => setAccessToken(e.target.value)} autoFocus />
      </div>
      <div className="space-y-2">
        <Label>Refresh Token</Label>
        <PasswordInput value={refreshToken} onChange={(e) => setRefreshToken(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>{t('detail.region')}</Label>
        <RegionSelect value={region} onChange={setRegion} />
      </div>
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!accessToken.trim() || !refreshToken.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

// --- Codex credentials import ---
export function CodexImportFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importCodex)
  const [accessToken, setAccessToken] = useState('')
  const [refreshToken, setRefreshToken] = useState('')
  const [idToken, setIdToken] = useState('')

  if (m.isSuccess) return <Done onDone={onDone} label={t('accounts.testSuccess')} />

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({
      accessToken: accessToken.trim(),
      refreshToken: refreshToken.trim(),
      idToken: idToken.trim() || undefined,
    })
  }
  return (
    <form onSubmit={submit} className="space-y-4">
      <div className="space-y-2">
        <Label>Access Token</Label>
        <PasswordInput value={accessToken} onChange={(e) => setAccessToken(e.target.value)} autoFocus />
      </div>
      <div className="space-y-2">
        <Label>Refresh Token</Label>
        <PasswordInput value={refreshToken} onChange={(e) => setRefreshToken(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label>ID Token</Label>
        <PasswordInput value={idToken} onChange={(e) => setIdToken(e.target.value)} />
      </div>
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!accessToken.trim() || !refreshToken.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}
