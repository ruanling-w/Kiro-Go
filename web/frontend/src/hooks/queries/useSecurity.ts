// Security-domain read hooks: trust-proxy setting + blocked-IP list.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { getSecuritySettings, listBlockedIps } from '@/services/security.service'

export function useSecuritySettings() {
  return useQuery({ queryKey: qk.security, queryFn: getSecuritySettings })
}

export function useBlockedIps() {
  return useQuery({ queryKey: qk.blockedIps, queryFn: listBlockedIps })
}
