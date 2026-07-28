// Provider model catalog + per-account models. Catalog is static-ish → cache long.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { getProviderModels } from '@/services/providers.service'
import { getAccountModels } from '@/services/accounts.service'

export function useProviderModels(provider: string, enabled = true) {
  return useQuery({
    queryKey: qk.providerModels(provider),
    queryFn: () => getProviderModels(provider),
    enabled: enabled && !!provider,
    staleTime: 5 * 60_000,
  })
}

export function useAccountModels(id: string, enabled = true) {
  return useQuery({
    queryKey: qk.accountModels(id),
    queryFn: () => getAccountModels(id),
    enabled: enabled && !!id,
    staleTime: 60_000,
  })
}
