// Settings-domain read hooks. Config rarely changes → generous staleTime.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  getSettings,
  getThinking,
  getEndpoint,
  getProxy,
  getPromptFilter,
  getTelegram,
} from '@/services/settings.service'

export function useSettings() {
  return useQuery({ queryKey: qk.settings, queryFn: getSettings })
}
export function useThinking() {
  return useQuery({ queryKey: qk.thinking, queryFn: getThinking })
}
export function useEndpoint() {
  return useQuery({ queryKey: qk.endpoint, queryFn: getEndpoint })
}
export function useProxy() {
  return useQuery({ queryKey: qk.proxy, queryFn: getProxy })
}
export function usePromptFilter() {
  return useQuery({ queryKey: qk.promptFilter, queryFn: getPromptFilter })
}
export function useTelegram() {
  return useQuery({ queryKey: qk.telegram, queryFn: getTelegram })
}
