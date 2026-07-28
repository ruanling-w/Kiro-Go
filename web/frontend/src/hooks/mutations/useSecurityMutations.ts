// Security-domain mutations: trust-proxy setting + blocked-IP list.
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  updateSecuritySettings,
  blockIp,
  unblockIp,
} from '@/services/security.service'
import type { SecuritySettings } from '@/types/settings'

export function useUpdateSecuritySettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: Partial<SecuritySettings>) => updateSecuritySettings(patch),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.security }),
  })
}

export function useBlockIp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ip, reason }: { ip: string; reason?: string }) => blockIp(ip, reason),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.blockedIps }),
  })
}

export function useUnblockIp() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ip: string) => unblockIp(ip),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.blockedIps }),
  })
}
