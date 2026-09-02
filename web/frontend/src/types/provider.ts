// Provider model catalog. GET /providers/{provider}/models.

export type ProviderKey =
  | 'kiro'
  | 'antigravity'
  | 'grok'
  | 'codex'
  | 'remotekiro'
  | 'voyage'

export interface ProviderModel {
  id: string
  name: string
  description: string
  supports_image: boolean
}

export interface ProviderModelsResponse {
  success: boolean
  provider: string
  source: 'cache' | 'static' | 'remote' | 'empty' | 'fallback'
  models: ProviderModel[]
}

// Live per-account model list (richer shape than cached string[]).
export interface AccountModelInfo {
  modelId: string
  modelName: string
  description: string
  inputTypes?: string[]
}
