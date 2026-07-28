// Status/stats read hooks. Overview polls status every 10s (see plan).
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { getStatus, getStats, getVersion } from '@/services/status.service'

export function useStatus() {
  return useQuery({
    queryKey: qk.status,
    queryFn: getStatus,
    refetchInterval: 10_000,
  })
}

export function useStats() {
  return useQuery({
    queryKey: qk.stats,
    queryFn: getStats,
    refetchInterval: 10_000,
  })
}

export function useVersion() {
  return useQuery({
    queryKey: qk.version,
    queryFn: getVersion,
    staleTime: Infinity,
  })
}
