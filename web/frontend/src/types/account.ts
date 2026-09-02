// Account types. NOTE the two shapes:
//  - AccountListItem: what GET /accounts returns (a bare array). Secrets are
//    masked; `hasToken` replaces the raw token. Includes runtime pool stats.
//  - AccountFull: what GET /accounts/{id}/full returns (bare object) — same
//    fields plus real secrets (accessToken/refreshToken/clientId/clientSecret),
//    no `hasToken`.

export type ProviderKey = 'kiro' | 'antigravity' | 'grok' | 'codex' | 'remotekiro' | 'voyage'

export interface VoyageQuotaBucket {
  model: string
  displayName?: string
  category?: string
  usedTokens: number
  freeLimitTokens: number
  remainingFreeTokens: number
  usedPercent: number
  costUsd: number
  pricePerMillion: number
  isFreeExhausted: boolean
}

export interface CodexQuotaWindow {
  key: string
  label?: string
  usedPct: number
  remaining?: number
  resetAt?: string
  limitHit?: boolean
}

export interface AccountListItem {
  id: string
  email: string
  userId: string
  nickname: string
  authMethod: string
  provider: string
  region: string
  enabled: boolean
  banStatus: string
  banReason: string
  banTime: string
  expiresAt: number
  hasToken: boolean
  remoteBaseURL: string
  remoteCheckKeyURL: string
  customModels?: string[]
  machineId: string
  weight: number

  overageStatus: string
  overageCapability: string
  overageCap: number
  overageRate: number
  currentOverages: number
  overageCheckedAt: number

  proxyURL: string
  subscriptionType: string
  subscriptionTitle: string

  daysRemaining: number
  usageCurrent: number
  usageLimit: number
  usagePercent: number
  nextResetDate: string
  lastRefresh: number

  trialUsageCurrent: number
  trialUsageLimit: number
  trialUsagePercent: number
  trialStatus: string
  trialExpiresAt: number

  agProjectId: string
  agTier: string
  agTierName: string
  agQuota: unknown

  grokAuthType: string
  codexAuthType?: string
  codexPlanType?: string
  codexAccountId?: string
  codexQuota?: CodexQuotaWindow[]
  codexLimitReached?: boolean
  codexResetCredits?: number

  voyageApiKey?: string
  voyageUsage?: Record<string, number>
  voyageQuota?: VoyageQuotaBucket[]

  // Runtime stats from the pool.
  requestCount: number
  errorCount: number
  totalTokens: number
  totalCredits: number
  lastUsed: number
}

export interface AccountFull extends Omit<AccountListItem, 'hasToken'> {
  accessToken: string
  refreshToken: string
  clientId: string
  clientSecret: string
}

/** PUT /accounts/{id} honors only these fields. */
export interface AccountUpdate {
  enabled?: boolean
  nickname?: string
  machineId?: string
  weight?: number
  proxyURL?: string
  customModels?: string[]
  remoteBaseURL?: string
  accessToken?: string
  apiKey?: string
  remoteCheckKeyURL?: string
}

export type BatchAction = 'enable' | 'disable' | 'refresh'

export interface BatchRequest {
  ids: string[]
  action: BatchAction
}

export interface ModelInfo {
  modelId: string
  modelName: string
  description: string
  inputTypes?: string[]
}
