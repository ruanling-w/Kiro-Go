// Kiro-SSO (2-leg M365) — pick region, then start → poll (loopback) with a
// manual-paste callback fallback.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useOAuthFlow } from '@/hooks/useOAuthFlow'
import {
  startKiroSso,
  pollKiroSso,
  completeKiroSso,
  cancelKiroSso,
} from '@/services/authFlows.service'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { RegionSelect } from '@/components/shared/RegionSelect'
import { OAuthFlowView } from '../OAuthFlowView'
import type { FlowComponentProps } from './types'

export function KiroSsoFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const [region, setRegion] = useState('us-east-1')
  const flow = useOAuthFlow<{ region: string }>({
    start: (args) => startKiroSso(args.region),
    poll: pollKiroSso,
    complete: completeKiroSso,
    cancel: cancelKiroSso,
  })

  if (flow.state.phase === 'idle') {
    return (
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>{t('detail.region')}</Label>
          <RegionSelect value={region} onChange={setRegion} />
        </div>
        <Button className="w-full" onClick={() => void flow.start({ region })}>
          {t('accounts.add')}
        </Button>
      </div>
    )
  }

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
