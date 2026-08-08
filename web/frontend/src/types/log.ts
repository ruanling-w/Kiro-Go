// Request log types. GET /logs returns {logs: RequestLog[]}. Timestamps are unix seconds.

export interface RequestLog {
  time: number
  endpoint: string
  model: string
  status: string
  error?: string
  errorType?: string
  tokens: number
  inputTokens?: number
  outputTokens?: number
  cacheReadTokens?: number
  cacheCreationTokens?: number
  cached?: boolean
  credits: number
  duration: number
  provider?: string
  accountId?: string
  clientIp?: string
  apiKeyId?: string
}
