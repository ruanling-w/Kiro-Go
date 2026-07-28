// security.service — blocked IPs + trust-proxy setting.
// GET /security/blocked-ips → {blockedIPs: [...]}; GET /security/settings → bare map.
import { http } from './httpClient'
import type { SecuritySettings, BlockedIPEntry } from '@/types/settings'
import type { SuccessEnvelope } from '@/types/common'

export function listBlockedIps(): Promise<BlockedIPEntry[]> {
  return http
    .get<{ blockedIPs: BlockedIPEntry[] }>('/security/blocked-ips')
    .then((r) => r.blockedIPs ?? [])
}

export function blockIp(ip: string, reason?: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/security/blocked-ips', { ip, reason })
}

export function unblockIp(ip: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/security/blocked-ips/unblock', { ip })
}

export function getSecuritySettings(): Promise<SecuritySettings> {
  return http.get<SecuritySettings>('/security/settings')
}

export function updateSecuritySettings(patch: Partial<SecuritySettings>): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/security/settings', patch)
}
