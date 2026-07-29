// Live request-log stream over SSE (GET /admin/api/logs/stream). The session
// cookie rides along same-origin (EventSource sends credentials by default);
// it's a GET so no CSRF token is needed. On connect the server sends a
// "snapshot" event (full ring buffer, newest-first) then a "log" event per new
// entry. We keep a capped buffer (matches the backend's 500-entry ring) so the
// list can't grow without bound.
//
// paused=true tears the connection down (no further events); flipping back to
// false reopens it and re-fetches a fresh snapshot.
import { useEffect, useRef, useState, useCallback } from 'react'
import type { RequestLog } from '@/types/log'

const STREAM_URL = '/admin/api/logs/stream'
const MAX_BUFFER = 500

export type StreamStatus = 'connecting' | 'live' | 'error'

// RequestLog has no stable id (only `time` in seconds, which collides under
// load), so we tag each row with a monotonic client-side key for React lists.
export interface LiveLog extends RequestLog {
  _key: number
}

export interface LogStream {
  logs: LiveLog[]
  status: StreamStatus
  clear: () => void
}

export function useLogStream(paused: boolean): LogStream {
  const [logs, setLogs] = useState<LiveLog[]>([])
  const [status, setStatus] = useState<StreamStatus>('connecting')
  const keyRef = useRef(0)

  const nextKey = useCallback(() => {
    keyRef.current += 1
    return keyRef.current
  }, [])

  const clear = useCallback(() => setLogs([]), [])

  useEffect(() => {
    if (paused) return

    setStatus('connecting')
    const es = new EventSource(STREAM_URL, { withCredentials: true })

    es.addEventListener('snapshot', (e) => {
      setStatus('live')
      try {
        const arr = JSON.parse((e as MessageEvent).data) as RequestLog[]
        setLogs(arr.map((log) => ({ ...log, _key: nextKey() })))
      } catch {
        // ignore malformed snapshot
      }
    })

    es.addEventListener('log', (e) => {
      try {
        const log = JSON.parse((e as MessageEvent).data) as RequestLog
        setLogs((prev) => {
          const next = [{ ...log, _key: nextKey() }, ...prev]
          return next.length > MAX_BUFFER ? next.slice(0, MAX_BUFFER) : next
        })
      } catch {
        // ignore malformed entry
      }
    })

    es.onopen = () => setStatus('live')
    es.onerror = () => setStatus('error') // EventSource auto-reconnects

    return () => es.close()
  }, [paused, nextKey])

  return { logs, status, clear }
}
