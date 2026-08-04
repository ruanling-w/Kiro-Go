// publicApi.service — session-free client endpoints (/v1/models, …).
// Uses bare axios (not admin httpClient) per AGENT.md §4.
import axios from 'axios'
import type { PublicModelsResponse } from '@/types/publicApi'

function normalizeBase(baseURL?: string): string {
  if (!baseURL) return ''
  return baseURL.replace(/\/+$/, '')
}

export async function listPublicModels(baseURL?: string): Promise<PublicModelsResponse> {
  const base = normalizeBase(baseURL)
  const url = base ? `${base}/v1/models` : '/v1/models'
  const res = await axios.get<PublicModelsResponse>(url, {
    // Cross-origin base (user typed a remote gateway) — no credentials needed.
    withCredentials: false,
    timeout: 15_000,
  })
  const data = res.data
  return {
    object: data?.object ?? 'list',
    data: Array.isArray(data?.data) ? data.data : [],
  }
}
