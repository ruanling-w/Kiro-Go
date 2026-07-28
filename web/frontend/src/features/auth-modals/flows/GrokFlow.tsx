// Grok OAuth — loopback poll + manual-paste fallback. No args to start.
import { useEffect } from 'react'
import { useOAuthFlow } from '@/hooks/useOAuthFlow'
import { startGrok, pollGrok, completeGrok, cancelGrok } from '@/services/authFlows.service'
import { OAuthFlowView } from '../OAuthFlowView'
import type { FlowComponentProps } from './types'

export function GrokFlow({ onDone }: FlowComponentProps) {
  const flow = useOAuthFlow<void>({
    start: startGrok,
    poll: pollGrok,
    complete: completeGrok,
    cancel: cancelGrok,
  })

  useEffect(() => {
    void flow.start()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
