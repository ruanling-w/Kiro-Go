// Antigravity OAuth — loopback poll + manual-paste fallback. No args to start.
import { useEffect } from 'react'
import { useOAuthFlow } from '@/hooks/useOAuthFlow'
import {
  startAntigravity,
  pollAntigravity,
  completeAntigravity,
  cancelAntigravity,
} from '@/services/authFlows.service'
import { OAuthFlowView } from '../OAuthFlowView'
import type { FlowComponentProps } from './types'

export function AntigravityFlow({ onDone }: FlowComponentProps) {
  const flow = useOAuthFlow<void>({
    start: startAntigravity,
    poll: pollAntigravity,
    complete: completeAntigravity,
    cancel: cancelAntigravity,
  })

  useEffect(() => {
    void flow.start()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
