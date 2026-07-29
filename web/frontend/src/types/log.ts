// Request log types. GET /logs returns {logs: RequestLog[]}. Timestamps are unix seconds.

export interface RequestLog {
  time: number
  endpoint: string
  model: string
  status: string
  error?: string
  errorType?: string
  tokens: number
  credits: number
  duration: number
  provider?: string
  accountId?: string
  clientIp?: string
  apiKeyId?: string
}
