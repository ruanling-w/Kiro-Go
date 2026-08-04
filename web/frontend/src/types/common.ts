// Shared envelope + status/stats shapes.

// Most mutations wrap their result in this. A few endpoints return bare
// objects/arrays instead (GET /accounts, GET /accounts/{id}/full, some config
// GETs) — those are typed directly, not via this envelope.
export interface SuccessEnvelope {
  success: boolean
}

// GET /status — also the legacy auth probe. Live snapshot.
export interface StatusSnapshot {
  version: string
  accounts: number
  available: number
  totalRequests: number
  successRequests: number
  failedRequests: number
  totalTokens: number
  totalCredits: number
  totalRpm?: number
  uptime: number
}

// GET /stats — aggregate counters (subset of status).
export interface StatsSnapshot {
  totalRequests: number
  successRequests: number
  failedRequests: number
  totalTokens: number
  totalCredits: number
  uptime: number
}
