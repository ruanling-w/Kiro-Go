// Codex OAuth — loopback poll + manual-paste fallback. No args to start.
import { useEffect } from 'react'
import { useOAuthFlow } from '@/hooks/useOAuthFlow'
import { startCodex, pollCodex, completeCodex, cancelCodex } from '@/services/authFlows.service'
import { OAuthFlowView } from '../OAuthFlowView'
import type { FlowComponentProps } from './types'

export function CodexFlow({ onDone }: FlowComponentProps) {
  const flow = useOAuthFlow<void>({
    start: startCodex,
    poll: pollCodex,
    complete: completeCodex,
    cancel: cancelCodex,
  })

  useEffect(() => {
    void flow.start()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return <OAuthFlowView flow={flow} onDone={onDone} allowManual />
}
