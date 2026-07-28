// status.service — live status/stats/version snapshots.
import { http } from './httpClient'
import type { StatusSnapshot, StatsSnapshot } from '@/types/common'

export function getStatus(): Promise<StatusSnapshot> {
  return http.get<StatusSnapshot>('/status')
}

export function getStats(): Promise<StatsSnapshot> {
  return http.get<StatsSnapshot>('/stats')
}

export interface VersionInfo {
  version: string
  latest?: string
  hasUpdate?: boolean
}

export function getVersion(): Promise<VersionInfo> {
  return http.get<VersionInfo>('/version')
}
