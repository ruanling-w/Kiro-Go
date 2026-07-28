// Log mutations: clear all logs. Invalidates the logs query.
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { clearLogs } from '@/services/logs.service'

export function useClearLogs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => clearLogs(),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.logs }),
  })
}
