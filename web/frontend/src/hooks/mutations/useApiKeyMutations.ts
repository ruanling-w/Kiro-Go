// API key mutations. All invalidate the api-keys list.
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  createApiKey,
  updateApiKey,
  deleteApiKey,
  resetApiKeyUsage,
} from '@/services/apikeys.service'
import type { ApiKeyCreate, ApiKeyUpdate } from '@/types/apikey'

export function useCreateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: ApiKeyCreate) => createApiKey(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.apiKeys }),
  })
}

export function useUpdateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: ApiKeyUpdate }) => updateApiKey(id, patch),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.apiKeys }),
  })
}

export function useDeleteApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteApiKey(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.apiKeys }),
  })
}

export function useResetApiKeyUsage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => resetApiKeyUsage(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.apiKeys }),
  })
}
