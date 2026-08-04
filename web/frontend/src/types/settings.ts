// Config/settings types. Most config GETs return bare field maps; POSTs use
// pointer fields (send only what changes) and return {success:true}.

export interface StatusSnapshot {
  version: string
  accounts: number
  available: number
  totalRequests: number
  successRequests: number
  failedRequests: number
  totalTokens: number
  totalCredits: number
  uptime: number
}

export interface Stats {
  totalRequests: number
  successRequests: number
  failedRequests: number
  totalTokens: number
  totalCredits: number
  uptime: number
}

export interface CoreSettings {
  apiKey: string
  requireApiKey: boolean
  port: number
  host: string
  allowOverUsage: boolean
  defaultApiKeyMultiplier: number
  defaultApiKeyRpmLimit: number
}

// POST /settings — omit a field to leave it unchanged. password change revokes sessions.
export interface CoreSettingsUpdate {
  apiKey?: string
  requireApiKey?: boolean
  password?: string
  allowOverUsage?: boolean
  defaultApiKeyMultiplier?: number
  defaultApiKeyRpmLimit?: number
}

export type ThinkingFormat = 'reasoning_content' | 'thinking' | 'think'

export interface ThinkingConfig {
  suffix: string
  openaiFormat: ThinkingFormat
  claudeFormat: ThinkingFormat
}

export type PreferredEndpoint = 'auto' | 'kiro' | 'codewhisperer' | 'amazonq'

export interface EndpointConfig {
  preferredEndpoint: PreferredEndpoint
  endpointFallback: boolean
}

export interface ProxyConfig {
  proxyURL: string
}

export interface PromptFilterRule {
  type: 'regex' | 'contains'
  pattern: string
  enabled?: boolean
}

export interface PromptFilterConfig {
  filterClaudeCode: boolean
  filterEnvNoise: boolean
  filterStripBoundaries: boolean
  rules: PromptFilterRule[]
}

export interface TelegramConfig {
  enabled: boolean
  chatId: string
  botTokenSet: boolean
  botTokenMasked: string
}

export interface TelegramUpdate {
  enabled?: boolean
  botToken?: string
  chatId?: string
}

export interface SecuritySettings {
  trustProxyHeaders: boolean
}

export interface BlockedIPEntry {
  ip: string
  reason: string
  blockedAt: number
}
