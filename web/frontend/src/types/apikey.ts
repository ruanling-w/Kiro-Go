// API key types. GET /api-keys returns {apiKeys: ApiKeyView[]}; GET /api-keys/{id}
// returns a bare ApiKeyView. Mutations wrap {success, apiKey}.

export interface ApiKeyView {
  id: string
  name: string
  keyMasked: string
  enabled: boolean
  migrated: boolean
  createdAt: number
  lastUsedAt: number
  tokenLimit: number
  creditLimit: number
  tokensUsed: number
  creditsUsed: number
  requestsCount: number
  expiresAt: number
  expired: boolean
  uniqueIps: number
  rpm: number
}

/** POST /api-keys — all optional; key auto-generated if empty; expiresAt unix seconds, 0 = never. */
export interface ApiKeyCreate {
  name?: string
  key?: string
  enabled?: boolean
  tokenLimit?: number
  creditLimit?: number
  expiresAt?: number
}

/** PUT /api-keys/{id} — omit a field to leave it unchanged. */
export interface ApiKeyUpdate {
  name?: string
  key?: string
  enabled?: boolean
  tokenLimit?: number
  creditLimit?: number
  expiresAt?: number
}

export interface ApiKeyCreateResponse {
  success: boolean
  id: string
  key: string // cleartext, shown ONCE
  apiKey: ApiKeyView
}

export interface ApiKeyRevealResponse {
  success: boolean
  id: string
  key: string
}

export interface KeyIPStat {
  ip: string
  requests: number
  lastSeen: number
}

export interface ApiKeyIPsResponse {
  ips: KeyIPStat[]
  uniqueCount: number
  rpm: number
}
