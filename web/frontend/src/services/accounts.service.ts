// accounts.service — every /accounts endpoint. Pure API layer (no React).
//
// Envelope is NOT uniform (see plan): GET /accounts returns a bare array; GET
// /accounts/{id}/full returns a bare object; mutations wrap {success,...}.
import { http } from './httpClient'
import type {
  AccountListItem,
  AccountFull,
  AccountUpdate,
  BatchRequest,
  ModelInfo,
} from '@/types/account'
import type { SuccessEnvelope } from '@/types/common'

export function listAccounts(): Promise<AccountListItem[]> {
  return http.get<AccountListItem[]>('/accounts')
}

export function getAccountFull(id: string): Promise<AccountFull> {
  return http.get<AccountFull>(`/accounts/${encodeURIComponent(id)}/full`)
}

export function updateAccount(id: string, patch: AccountUpdate): Promise<SuccessEnvelope> {
  return http.put<SuccessEnvelope>(`/accounts/${encodeURIComponent(id)}`, patch)
}

export function deleteAccount(id: string): Promise<SuccessEnvelope> {
  return http.delete<SuccessEnvelope>(`/accounts/${encodeURIComponent(id)}`)
}

export function batchAccounts(body: BatchRequest): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/accounts/batch', body)
}

export function refreshAccount(id: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>(`/accounts/${encodeURIComponent(id)}/refresh`)
}

export function refreshAccountModels(id: string): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>(`/accounts/${encodeURIComponent(id)}/models/refresh`)
}

export function refreshAllAccountsModels(): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/accounts/models/refresh')
}

export interface TestAccountResult {
  success: boolean
  model?: string
  message?: string
  error?: string
  latency?: number
}

export function testAccount(id: string, model?: string): Promise<TestAccountResult> {
  return http.post<TestAccountResult>(`/accounts/${encodeURIComponent(id)}/test`, { model })
}

export function getAccountModels(id: string): Promise<ModelInfo[]> {
  return http.get<{ models?: ModelInfo[] } | ModelInfo[]>(
    `/accounts/${encodeURIComponent(id)}/models`,
  ).then((r) => (Array.isArray(r) ? r : (r.models ?? [])))
}

export function getAccountModelsCached(id: string): Promise<ModelInfo[]> {
  return http.get<{ models?: ModelInfo[] } | ModelInfo[]>(
    `/accounts/${encodeURIComponent(id)}/models/cached`,
  ).then((r) => (Array.isArray(r) ? r : (r.models ?? [])))
}

export interface OverageState {
  overageStatus: string
  overageCapability: string
  overageCap: number
  overageRate: number
  currentOverages: number
  overageCheckedAt: number
}

export function getAccountOverage(id: string): Promise<OverageState> {
  return http.get<OverageState>(`/accounts/${encodeURIComponent(id)}/overage`)
}

export function setAccountOverage(id: string, enabled: boolean): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>(`/accounts/${encodeURIComponent(id)}/overage`, { enabled })
}

export interface CodexResetCreditItem {
  id: string
  reset_type: string
  status: string
  granted_at: string
  expires_at: string
  title: string
  description: string
  profile_image_url?: string
}

export interface CodexResetCreditsResponse {
  credits: CodexResetCreditItem[]
  available_count: number
}

export function getCodexResetCredits(id: string): Promise<CodexResetCreditsResponse> {
  return http.get<CodexResetCreditsResponse>(`/accounts/${encodeURIComponent(id)}/codex/reset-credits`)
}

export function consumeCodexResetCredit(id: string, creditId?: string): Promise<SuccessEnvelope & { creditId?: string; usage?: any }> {
  return http.post<SuccessEnvelope & { creditId?: string; usage?: any }>(
    `/accounts/${encodeURIComponent(id)}/codex/reset-credits/consume`,
    { credit_id: creditId },
  )
}

export function generateMachineId(): Promise<{ machineId: string }> {
  return http.get<{ machineId: string }>('/generate-machine-id')
}

