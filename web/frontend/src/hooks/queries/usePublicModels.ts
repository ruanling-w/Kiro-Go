// Public /v1/models catalog — no admin session. Keyed by base URL so changing
// the docs base refetches against the new origin.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { listPublicModels } from '@/services/publicApi.service'

export function usePublicModels(baseURL: string) {
  const base = baseURL.replace(/\/+$/, '')
  return useQuery({
    queryKey: qk.publicModels(base),
    queryFn: () => listPublicModels(base || undefined),
    staleTime: 5 * 60_000,
    retry: false,
  })
}
