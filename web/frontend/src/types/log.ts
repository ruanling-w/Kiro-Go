// Request log types. GET /logs returns {logs: RequestLog[]}. Timestamps are unix seconds.

export interface RequestLog {
  time: number
  endpoint: string
  model: string
  status: string
  errorType?: string
  tokens: number
  credits: number
  duration: number
  provider?: string
  account?: string
  ip?: string
  apiKey?: string
  apiKeyId?: string
}
