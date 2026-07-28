// settings.service — all config domains. GETs return bare field maps; POSTs
// take pointer-style partials (send only what changes) and return {success}.
import { http } from './httpClient'
import type {
  CoreSettings,
  CoreSettingsUpdate,
  ThinkingConfig,
  EndpointConfig,
  ProxyConfig,
  PromptFilterConfig,
  TelegramConfig,
  TelegramUpdate,
} from '@/types/settings'
import type { SuccessEnvelope } from '@/types/common'

export function getSettings(): Promise<CoreSettings> {
  return http.get<CoreSettings>('/settings')
}
export function updateSettings(patch: CoreSettingsUpdate): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/settings', patch)
}

export function getThinking(): Promise<ThinkingConfig> {
  return http.get<ThinkingConfig>('/thinking')
}
export function updateThinking(body: ThinkingConfig): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/thinking', body)
}

export function getEndpoint(): Promise<EndpointConfig> {
  return http.get<EndpointConfig>('/endpoint')
}
export function updateEndpoint(body: EndpointConfig): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/endpoint', body)
}

export function getProxy(): Promise<ProxyConfig> {
  return http.get<ProxyConfig>('/proxy')
}
export function updateProxy(body: ProxyConfig): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/proxy', body)
}

export function getPromptFilter(): Promise<PromptFilterConfig> {
  return http.get<PromptFilterConfig>('/prompt-filter')
}
export function updatePromptFilter(body: PromptFilterConfig): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/prompt-filter', body)
}

export function getTelegram(): Promise<TelegramConfig> {
  return http.get<TelegramConfig>('/telegram')
}
export function updateTelegram(patch: TelegramUpdate): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/telegram', patch)
}
export function testTelegram(): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/telegram/test')
}

export function resetStats(): Promise<SuccessEnvelope> {
  return http.post<SuccessEnvelope>('/stats/reset')
}
