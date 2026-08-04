// WebCookieFlow — "Kiro Web cookie" import (legacy method id `cookie`).
//
// Rebuilds modalCookie + importFromCookie from web/js/auth-modals.js. The user
// signs in at app.kiro.dev in a normal browser and copies the `RefreshToken`
// cookie out of devtools; the server exchanges it for an access token on import.
// Always a social account (Google/Github), so no clientId/clientSecret is needed.
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
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
import { HelpBlock, Steps } from './HelpBlock'
import { Done, ErrorNote, useImport } from './importShared'

export function WebCookieFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const m = useImport(importCredentials)
  const [provider, setProvider] = useState('Google')
  const [refreshToken, setRefreshToken] = useState('')

  if (m.isSuccess) return <Done onDone={onDone} label={t('cookie.importSuccess')} />

  function submit(e: FormEvent) {
    e.preventDefault()
    m.mutate({
      refreshToken: refreshToken.trim(),
      // Empty accessToken on purpose: the import path refreshes to obtain one.
      accessToken: '',
      clientId: '',
      clientSecret: '',
      authMethod: 'social',
      provider,
    })
  }

  const link = t('cookie.link')

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-muted-foreground">{t('modal.cookieDesc')}</p>

      <HelpBlock title={t('cookie.howToGet')}>
        <Steps>
          <li>
            {t('cookie.step1')}{' '}
            <a
              href={link}
              target="_blank"
              rel="noreferrer noopener"
              className="text-primary underline break-all"
            >
              {link}
            </a>
          </li>
          <li>{t('cookie.step2')}</li>
          <li>{t('cookie.step3')}</li>
        </Steps>
      </HelpBlock>

      <div className="space-y-2">
        <Label>{t('cookie.provider')}</Label>
        <Select value={provider} onValueChange={setProvider}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="Google">{t('cookie.google')}</SelectItem>
            <SelectItem value="Github">{t('cookie.github')}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>{t('cookie.refreshToken')}</Label>
        <textarea
          value={refreshToken}
          onChange={(e) => setRefreshToken(e.target.value)}
          placeholder={t('cookie.refreshTokenPlaceholder')}
          autoFocus
          className="min-h-20 w-full rounded-lg border bg-transparent px-3 py-2 font-mono text-xs"
        />
      </div>

      <ErrorNote error={m.error} />
      <Button type="submit" className="w-full" disabled={!refreshToken.trim() || m.isPending}>
        {m.isPending ? <HamsterWheel size="sm" /> : t('accounts.add')}
      </Button>
    </form>
  )
}
