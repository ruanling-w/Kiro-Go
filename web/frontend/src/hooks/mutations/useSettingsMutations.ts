// Settings-domain mutations. Each invalidates its own config key (and status,
// since a password change revokes sessions and settings affect the snapshot).
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  updateSettings,
  updateThinking,
  updateEndpoint,
  updateProxy,
  updatePromptFilter,
  updateTelegram,
  testTelegram,
  resetStats,
} from '@/services/settings.service'
import type {
  CoreSettingsUpdate,
  ThinkingConfig,
  EndpointConfig,
  ProxyConfig,
  PromptFilterConfig,
  TelegramUpdate,
} from '@/types/settings'

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: CoreSettingsUpdate) => updateSettings(patch),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.settings })
      void qc.invalidateQueries({ queryKey: qk.status })
    },
  })
}

export function useUpdateThinking() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: ThinkingConfig) => updateThinking(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.thinking }),
  })
}

export function useUpdateEndpoint() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: EndpointConfig) => updateEndpoint(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.endpoint }),
  })
}

export function useUpdateProxy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: ProxyConfig) => updateProxy(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.proxy }),
  })
}

export function useUpdatePromptFilter() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: PromptFilterConfig) => updatePromptFilter(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.promptFilter }),
  })
}

export function useUpdateTelegram() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: TelegramUpdate) => updateTelegram(patch),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.telegram }),
  })
}

export function useTestTelegram() {
  return useMutation({ mutationFn: () => testTelegram() })
}

export function useResetStats() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => resetStats(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: qk.stats })
      void qc.invalidateQueries({ queryKey: qk.status })
    },
  })
}
