// API keys read hooks.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { listApiKeys, getApiKeyIPs } from '@/services/apikeys.service'

export function useApiKeys() {
  return useQuery({
    queryKey: qk.apiKeys,
    queryFn: listApiKeys,
    // Live RPM is a 60s RAM window — poll so the table/dashboard stay fresh.
    refetchInterval: 5_000,
  })
}

export function useApiKeyIPs(id: string | null) {
  return useQuery({
    queryKey: id ? qk.apiKeyIps(id) : ['api-keys', 'ips', 'none'],
    queryFn: () => getApiKeyIPs(id as string),
    enabled: !!id,
    refetchInterval: id ? 5_000 : false,
  })
}
