// Antigravity OAuth — loopback poll + manual-paste fallback. No args to start.
import { useEffect, useRef } from 'react'
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

  // Guard against StrictMode's double-invoked effect: starting twice would open
  // two tabs and mint two PKCE verifiers, so the code from one session gets
  // exchanged against the other's verifier → PKCE mismatch.
  const started = useRef(false)
  useEffect(() => {
    if (started.current) return
    started.current = true
    void flow.start()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
