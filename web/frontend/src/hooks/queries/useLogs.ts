// Logs read hook. Polls every 5s when auto-refresh is enabled (see plan).
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { listLogs } from '@/services/logs.service'

export function useLogs(autoRefresh: boolean) {
  return useQuery({
    queryKey: qk.logs,
    queryFn: listLogs,
    refetchInterval: autoRefresh ? 5_000 : false,
  })
}
