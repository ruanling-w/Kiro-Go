// LocalCacheFlow — "Kiro local cache" import (legacy method id `local`).
//
// Rebuilds modalLocal + importLocalKiro from web/js/auth-modals.js, which the
// React rewrite dropped entirely. The user pastes (or uploads) the two JSON files
// the AWS SSO CLI cache leaves on THEIR machine — ~/.aws/sso/cache/ — so this
// works fine against a remote deployment; nothing is read server-side.
//
// Social providers (Google/Github) only need kiro-auth-token.json. IdC providers
// (BuilderId/Enterprise) additionally need the {hash}.json holding the OAuth
// client registration, because refreshing an IdC token requires clientId +
// clientSecret. Posts to the existing /auth/credentials endpoint.
import { useState, type ChangeEvent, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Upload } from 'lucide-react'
import { importCredentials } from '@/services/authFlows.service'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import type { FlowComponentProps } from './types'
import { HelpBlock, Code } from './HelpBlock'
import { Done, ErrorNote, useImport } from './importShared'

const PROVIDERS = [
  { value: 'BuilderId', labelKey: 'local.providerBuilderId' },
  { value: 'Enterprise', labelKey: 'local.providerEnterprise' },
  { value: 'Google', labelKey: 'local.providerGoogle' },
  { value: 'Github', labelKey: 'local.providerGithub' },
] as const

export function LocalCacheFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importCredentials)
  const [provider, setProvider] = useState<string>('BuilderId')
  const [tokenJson, setTokenJson] = useState('')
  const [clientJson, setClientJson] = useState('')
  const [localError, setLocalError] = useState('')

  // Google/Github tokens refresh through the social endpoint — no client file.
  const isSocial = provider === 'Google' || provider === 'Github'

  if (m.isSuccess) return <Done onDone={onDone} label={t('local.importSuccess')} />

  function loadFile(e: ChangeEvent<HTMLInputElement>, set: (v: string) => void) {
    const file = e.target.files?.[0]
    if (!file) return
    void file.text().then(set)
    e.target.value = '' // allow re-picking the same file
  }

  function submit(e: FormEvent) {
    e.preventDefault()
    setLocalError('')

    const rawToken = tokenJson.trim()
    if (!rawToken) return setLocalError(t('local.tokenMissing'))

    let tokenData: Record<string, unknown>
    try {
      tokenData = JSON.parse(rawToken)
    } catch {
      return setLocalError(t('local.tokenInvalid'))
    }
    const refreshToken = typeof tokenData.refreshToken === 'string' ? tokenData.refreshToken : ''
    if (!refreshToken) return setLocalError(t('local.refreshTokenMissing'))

    let clientId = ''
    let clientSecret = ''
    if (!isSocial) {
      const rawClient = clientJson.trim()
      if (!rawClient) return setLocalError(t('local.clientMissing'))
      let clientData: Record<string, unknown>
      try {
        clientData = JSON.parse(rawClient)
      } catch {
        return setLocalError(t('local.clientInvalid'))
      }
      clientId = typeof clientData.clientId === 'string' ? clientData.clientId : ''
      clientSecret = typeof clientData.clientSecret === 'string' ? clientData.clientSecret : ''
      if (!clientId || !clientSecret) return setLocalError(t('local.clientSecretMissing'))
    }

    m.mutate({
      refreshToken,
      // Legacy flow deliberately sent a possibly-empty accessToken and let the
      // server-side refresh mint a fresh one.
      accessToken: typeof tokenData.accessToken === 'string' ? tokenData.accessToken : '',
      clientId,
      clientSecret,
      region: typeof tokenData.region === 'string' ? tokenData.region : '',
      authMethod: isSocial ? 'social' : 'idc',
      provider,
    })
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">{t('modal.localDesc')}</p>

      <HelpBlock title={t('local.fileLocation')}>
        <p>
          {t('local.windows')}: <Code>%USERPROFILE%\.aws\sso\cache\</Code>
        </p>
        <p>
          {t('local.macosLinux')}: <Code>~/.aws/sso/cache/</Code>
        </p>
      </HelpBlock>

      <div className="space-y-2">
        <Label>{t('local.loginChannel')}</Label>
        <Select value={provider} onValueChange={setProvider}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PROVIDERS.map((p) => (
              <SelectItem key={p.value} value={p.value}>
                {t(p.labelKey)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <JsonField
        label={t('local.tokenFile')}
        hint={t('local.tokenRequired')}
        value={tokenJson}
        onChange={setTokenJson}
        onFile={(e) => loadFile(e, setTokenJson)}
        placeholder={t('local.pasteOrUpload')}
        uploadLabel={t('local.upload')}
        autoFocus
      />

      {!isSocial && (
        <JsonField
          label={t('local.clientFile')}
          hint={t('local.clientRequired')}
          value={clientJson}
          onChange={setClientJson}
          onFile={(e) => loadFile(e, setClientJson)}
          placeholder={t('local.pasteOrUpload')}
          uploadLabel={t('local.upload')}
        />
      )}

      {localError && (
        <p className="text-sm text-destructive" role="alert">
          {localError}
        </p>
      )}
      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!tokenJson.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}

interface JsonFieldProps {
  label: string
  hint: string
  value: string
  onChange: (v: string) => void
  onFile: (e: ChangeEvent<HTMLInputElement>) => void
  placeholder: string
  uploadLabel: string
  autoFocus?: boolean
}

/** Textarea + "upload" file picker pair, mirroring the legacy `.input-row`. */
function JsonField({
  label,
  hint,
  value,
  onChange,
  onFile,
  placeholder,
  uploadLabel,
  autoFocus,
}: JsonFieldProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-baseline justify-between gap-2">
        <Label>{label}</Label>
        <span className="text-xs text-muted-foreground">{hint}</span>
      </div>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className="min-h-24 w-full rounded-lg border bg-transparent px-3 py-2 font-mono text-xs"
      />
      <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs hover:bg-muted">
        <Upload className="size-3.5" />
        {uploadLabel}
        <input type="file" accept=".json,application/json" onChange={onFile} className="hidden" />
      </label>
    </div>
  )
}
