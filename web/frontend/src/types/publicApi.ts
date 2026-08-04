// Public (non-admin) API shapes — /v1/models is session-free and returns an
// OpenAI-style list envelope. Combo entries add combo/strategy flags on top of
// the base model info from buildModelInfo.

export interface PublicModelCapabilities {
  vision?: boolean
  image?: boolean
  image_vision?: boolean
  combo?: boolean
}

export interface PublicModel {
  id: string
  object?: string
  created?: number
  owned_by?: string
  supports_image?: boolean
  input_modalities?: string[]
  modalities?: { input?: string[]; output?: string[] }
  capabilities?: PublicModelCapabilities
  combo?: boolean
  strategy?: string
}

export interface PublicModelsResponse {
  object: string
  data: PublicModel[]
}
