export type ComboStrategy = 'fallback' | 'round-robin' | 'fusion'

export interface ComboModel {
  model: string
  position: number
}

export interface Combo {
  id: string
  name: string
  strategy: ComboStrategy
  stickyLimit: number
  fusionQuorum?: number
  fusionTimeoutMs?: number
  judgeModel?: string
  revision: number
  createdAt: number
  updatedAt: number
  models: ComboModel[]
}

export interface ComboInput {
  name: string
  strategy: ComboStrategy
  stickyLimit: number
  revision?: number
  models: string[]
  fusionQuorum?: number
  fusionTimeoutMs?: number
  judgeModel?: string
}

export interface ComboListResponse { data: Combo[] }
export interface ComboResetResponse { ok: boolean }
export type ComboFieldErrors = Partial<Record<keyof ComboInput, string>>
