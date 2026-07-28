// apikeys.service — /api-keys CRUD + reveal/reset/ips.
// GET /api-keys → {apiKeys:[]}; GET /api-keys/{id} → bare view; mutations wrap.
import { http } from './httpClient'
import type {
  ApiKeyView,
  ApiKeyCreate,
  ApiKeyUpdate,
  ApiKeyCreateResponse,
  ApiKeyRevealResponse,
  ApiKeyIPsResponse,
} from '@/types/apikey'
import type { SuccessEnvelope } from '@/types/common'

export function listApiKeys(): Promise<ApiKeyView[]> {
  return http.get<{ apiKeys?: ApiKeyView[] }>('/api-keys').then((r) => r.apiKeys ?? [])
}

export function createApiKey(body: ApiKeyCreate): Promise<ApiKeyCreateResponse> {
  return http.post<ApiKeyCreateResponse>('/api-keys', body)
}

export function updateApiKey(id: string, patch: ApiKeyUpdate): Promise<SuccessEnvelope> {
  return http.put<SuccessEnvelope>(`/api-keys/${encodeURIComponent(id)}`, patch)
}

export function deleteApiKey(id: string): Promise<SuccessEnvelope> {
  return http.delete<SuccessEnvelope>(`/api-keys/${encodeURIComponent(id)}`)
}

export function resetApiKeyUsage(id: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>(`/api-keys/${encodeURIComponent(id)}/reset-usage`)
}

export function revealApiKey(id: string): Promise<ApiKeyRevealResponse> {
  return http.get<ApiKeyRevealResponse>(`/api-keys/${encodeURIComponent(id)}/reveal`)
}

export function getApiKeyIPs(id: string): Promise<ApiKeyIPsResponse> {
  return http.get<ApiKeyIPsResponse>(`/api-keys/${encodeURIComponent(id)}/ips`)
}
