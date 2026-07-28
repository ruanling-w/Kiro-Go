// logs.service — GET /logs returns {logs:[]}; DELETE /logs clears.
import { http } from './httpClient'
import type { RequestLog } from '@/types/log'
import type { SuccessEnvelope } from '@/types/common'

export function listLogs(): Promise<RequestLog[]> {
  return http.get<{ logs?: RequestLog[] }>('/logs').then((r) => r.logs ?? [])
}

export function clearLogs(): Promise<SuccessEnvelope> {
  return http.delete<SuccessEnvelope>('/logs')
}
