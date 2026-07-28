// checkKey.service — the PUBLIC key-lookup endpoint behind /check. No admin
// session/CSRF: it self-authenticates on the submitted key value. Uses a bare
// axios call (not the admin httpClient) since it lives outside /admin/api.
import axios from 'axios'

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

export async function lookupKey(key: string): Promise<CheckKeyResponse> {
  const res = await axios.post<CheckKeyResponse>('/check/api/lookup', { key })
  return res.data
}
