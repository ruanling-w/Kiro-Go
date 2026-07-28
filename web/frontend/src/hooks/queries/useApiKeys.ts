// API keys read hooks.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import { listApiKeys, getApiKeyIPs } from '@/services/apikeys.service'

export function useApiKeys() {
  return useQuery({
    queryKey: qk.apiKeys,
    queryFn: listApiKeys,
  })
}

export function useApiKeyIPs(id: string | null) {
  return useQuery({
    queryKey: id ? qk.apiKeyIps(id) : ['api-keys', 'ips', 'none'],
    queryFn: () => getApiKeyIPs(id as string),
    enabled: !!id,
  })
}
