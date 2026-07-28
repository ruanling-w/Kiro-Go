// authFlows.service — every add-account flow: OAuth (start/poll/complete/cancel)
// and direct imports. Pure API layer. Endpoints are stateful via a server-side
// sessionId (see plan). Request/response shapes are typed per-endpoint.
import { http } from './httpClient'
import type {
  StartResponse,
  PollResponse,
  CompleteResponse,
  BatchImportResponse,
} from '@/types/auth'
import type { SuccessEnvelope } from '@/types/common'

// --- Grok OAuth (loopback + manual paste fallback) ---
export function startGrok(): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/grok/start')
}
export function pollGrok(sessionId: string): Promise<PollResponse> {
  return http.post<PollResponse>('/auth/grok/poll', { sessionId })
}
export function completeGrok(sessionId: string, callbackUrl: string): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/grok/complete', { sessionId, callbackUrl })
}
export function cancelGrok(sessionId: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/auth/grok/cancel', { sessionId })
}

// --- Antigravity OAuth ---
export function startAntigravity(): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/antigravity/start')
}
export function pollAntigravity(sessionId: string): Promise<PollResponse> {
  return http.post<PollResponse>('/auth/antigravity/poll', { sessionId })
}
export function completeAntigravity(sessionId: string, callbackUrl: string): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/antigravity/complete', { sessionId, callbackUrl })
}
export function cancelAntigravity(sessionId: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/auth/antigravity/cancel', { sessionId })
}

// --- Codex OAuth ---
export function startCodex(): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/codex/start')
}
export function pollCodex(sessionId: string): Promise<PollResponse> {
  return http.post<PollResponse>('/auth/codex/poll', { sessionId })
}
export function completeCodex(sessionId: string, callbackUrl: string): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/codex/complete', { sessionId, callbackUrl })
}
export function cancelCodex(sessionId: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/auth/codex/cancel', { sessionId })
}

// --- Kiro-SSO OAuth (2-leg M365) ---
export function startKiroSso(region: string): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/kiro-sso/start', { region })
}
export function pollKiroSso(sessionId: string): Promise<PollResponse> {
  return http.post<PollResponse>('/auth/kiro-sso/poll', { sessionId })
}
export function completeKiroSso(sessionId: string, callbackUrl: string): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/kiro-sso/complete', { sessionId, callbackUrl })
}
export function cancelKiroSso(sessionId: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/auth/kiro-sso/cancel', { sessionId })
}

// --- BuilderID device-code (start → poll) ---
export function startBuilderId(region: string): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/builderid/start', { region })
}
export function pollBuilderId(sessionId: string): Promise<PollResponse> {
  return http.post<PollResponse>('/auth/builderid/poll', { sessionId })
}

// --- IAM SSO (start → complete via pasted callback) ---
export function startIamSso(startUrl: string, region: string): Promise<StartResponse> {
  return http.post<StartResponse>('/auth/iam-sso/start', { startUrl, region })
}
export function completeIamSso(sessionId: string, callbackUrl: string): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/iam-sso/complete', { sessionId, callbackUrl })
}

// --- Direct imports (no session) ---
export interface KiroApiKeyImport {
  apiKey: string
  region?: string
  nickname?: string
}
export function importKiroApiKey(body: KiroApiKeyImport): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/kiro-apikey', body)
}

export interface RemoteKiroImport {
  baseURL: string
  apiKey: string
  nickname?: string
  weight?: number
  proxyURL?: string
  checkKeyURL?: string
}
export function importRemoteKiro(body: RemoteKiroImport): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/remote-kiro', body)
}

export interface SsoTokenImport {
  bearerToken: string
  region?: string
}
export function importSsoToken(body: SsoTokenImport): Promise<BatchImportResponse> {
  return http.post<BatchImportResponse>('/auth/sso-token', body)
}

export interface CredentialsImport {
  accessToken: string
  refreshToken: string
  clientId?: string
  clientSecret?: string
  authMethod?: string
  provider?: string
  region?: string
}
export function importCredentials(body: CredentialsImport): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/auth/credentials', body)
}

export interface CodexImport {
  accessToken: string
  refreshToken: string
  idToken?: string
  expiresIn?: number
  weight?: number
}
export function importCodex(body: CodexImport): Promise<CompleteResponse> {
  return http.post<CompleteResponse>('/accounts/codex/import', body)
}
