// ApiEndpointsSection — merged from the old "API" view. Lists the copyable client
// endpoints the gateway exposes, plus a Models/Stats viewer dialog (raw JSON from
// the public /v1/models and admin /stats).
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CopyButton } from '@/components/shared/CopyButton'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useStats } from '@/hooks/queries/useStatus'
import { SettingsSection } from './SettingsSection'

interface Endpoint {
  method: string
  path: string
  desc: string
}

const ENDPOINTS: Endpoint[] = [
  { method: 'POST', path: '/v1/messages', desc: 'Anthropic Messages API' },
  { method: 'POST', path: '/v1/chat/completions', desc: 'OpenAI Chat Completions' },
  { method: 'GET', path: '/v1/models', desc: 'Model catalog' },
  { method: 'GET', path: '/v1/stats', desc: 'Usage stats' },
  { method: 'GET', path: '/version', desc: 'Version info' },
]

function origin(): string {
  return typeof window !== 'undefined' ? window.location.origin : ''
}

export function ApiEndpointsSection() {
  const { t } = useTranslation()
  const [viewStats, setViewStats] = useState(false)
  const stats = useStats()

  return (
    <SettingsSection id="api" title={t('nav.usage')} description={t('tabs.api')}>
      <ul className="divide-y">
        {ENDPOINTS.map((e) => {
          const full = origin() + e.path
          return (
            <li key={e.path} className="flex items-center gap-3 py-2.5">
              <span className="w-14 shrink-0 rounded bg-muted px-1.5 py-0.5 text-center text-xs font-semibold text-muted-foreground">
                {e.method}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate font-mono text-sm">{e.path}</p>
                <p className="truncate text-xs text-muted-foreground">{e.desc}</p>
              </div>
              <CopyButton value={full} label={t('common.copy')} />
            </li>
          )
        })}
      </ul>

      <Button variant="outline" size="sm" onClick={() => setViewStats(true)}>
        {t('settings.statistics')}
      </Button>

      <Dialog open={viewStats} onOpenChange={setViewStats}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('settings.statistics')}</DialogTitle>
          </DialogHeader>
          {stats.isPending ? (
            <HamsterLoader size="sm" />
          ) : (
            <pre className="max-h-[60vh] overflow-auto rounded-lg bg-muted p-4 text-xs">
              {JSON.stringify(stats.data, null, 2)}
            </pre>
          )}
        </DialogContent>
      </Dialog>
    </SettingsSection>
  )
}
