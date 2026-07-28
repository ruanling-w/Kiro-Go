// IAM Identity Center SSO — enter start URL + region, then start → complete via
// pasted callback URL (no polling; the admin pastes the redirect result).
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useOAuthFlow } from '@/hooks/useOAuthFlow'
import { startIamSso, completeIamSso } from '@/services/authFlows.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RegionSelect } from '@/components/shared/RegionSelect'
import { OAuthFlowView } from '../OAuthFlowView'
import type { FlowComponentProps } from './types'

export function IamSsoFlow({ onDone }: FlowComponentProps) {
  const { t } = useTranslation()
  const [startUrl, setStartUrl] = useState('')
  const [region, setRegion] = useState('us-east-1')
  const flow = useOAuthFlow<{ startUrl: string; region: string }>({
    start: (args) => startIamSso(args.startUrl, args.region),
    complete: completeIamSso,
  })

  if (flow.state.phase === 'idle') {
    return (
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>Start URL</Label>
          <Input
            value={startUrl}
            onChange={(e) => setStartUrl(e.target.value)}
            placeholder="https://d-xxxx.awsapps.com/start"
          />
        </div>
        <div className="space-y-2">
          <Label>{t('detail.region')}</Label>
          <RegionSelect value={region} onChange={setRegion} />
        </div>
        <Button
          className="w-full"
          disabled={!startUrl.trim()}
          onClick={() => void flow.start({ startUrl, region })}
        >
          {t('accounts.add')}
        </Button>
      </div>
    )
  }

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
