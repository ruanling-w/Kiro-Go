// providers.service — provider model catalog for the admin UI drill-down.
import { http } from './httpClient'
import type { ProviderModelsResponse } from '@/types/provider'

export function getProviderModels(provider: string): Promise<ProviderModelsResponse> {
  return http.get<ProviderModelsResponse>(
    `/providers/${encodeURIComponent(provider)}/models`,
  )
}
