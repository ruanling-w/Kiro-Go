// checkKey.service — the PUBLIC key-lookup endpoint behind /check. No admin
// session/CSRF: it self-authenticates on the submitted key value. Uses a bare
// axios call (not the admin httpClient) since it lives outside /admin/api.
import axios, { AxiosError } from 'axios'
import { ApiError } from '@/services/httpClient'

export interface CheckKeyLog {
  time: number
  endpoint: string
  model: string
  status: string
  errorType?: string
  tokens: number
  credits: number
  duration: number
}

export interface CheckKeyResponse {
  keyMasked: string
  name?: string
  enabled: boolean
  creditLimit: number
  creditsUsed: number
  creditsRemaining: number
  creditUnlimited: boolean
  tokenLimit: number
  tokensUsed: number
  tokensRemaining: number
  tokenUnlimited: boolean
  expiresAt: number
  neverExpires: boolean
  expired: boolean
  daysRemaining: number
  createdAt: number
  lastUsedAt?: number
  requestsCount: number
  logs: CheckKeyLog[]
}

/** Key is usable for new requests (enabled and not past expiry). */
export function isCheckKeyActive(data: CheckKeyResponse): boolean {
  return data.enabled && !data.expired
}

function errorMessage(err: unknown): string {
  if (err instanceof AxiosError) {
    const data = err.response?.data
    if (data && typeof data === 'object' && typeof (data as { error?: unknown }).error === 'string') {
      return (data as { error: string }).error
    }
    return err.message || 'Request failed'
  }
  if (err instanceof Error) return err.message
  return 'Request failed'
}

export async function lookupKey(key: string): Promise<CheckKeyResponse> {
  try {
    const res = await axios.post<CheckKeyResponse>('/check/api/lookup', { key })
    const data = res.data
    // Normalize optional arrays so UI never has to null-check.
    return { ...data, logs: data.logs ?? [] }
  } catch (err) {
    const status = err instanceof AxiosError ? (err.response?.status ?? 0) : 0
    throw new ApiError(status, errorMessage(err))
  }
}
