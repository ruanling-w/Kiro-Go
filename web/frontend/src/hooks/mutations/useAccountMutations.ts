// Account mutations. Each invalidates the accounts list (and related keys) so
// the UI reflects the change without a manual refetch.
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  updateAccount,
  deleteAccount,
  batchAccounts,
  refreshAccount,
  refreshAccountModels,
  refreshAllAccountsModels,
  setAccountOverage,
  consumeCodexResetCredit,
} from '@/services/accounts.service'
import type { AccountUpdate, BatchRequest } from '@/types/account'

export function useUpdateAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: AccountUpdate }) =>
      updateAccount(id, patch),
    onSuccess: (_d, { id }) => {
      void qc.invalidateQueries({ queryKey: qk.accounts })
      void qc.invalidateQueries({ queryKey: qk.accountFull(id) })
    },
  })
}

export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

export function useBatchAccounts() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: BatchRequest) => batchAccounts(body),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

export function useRefreshAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => refreshAccount(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

export function useRefreshAccountModels() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => refreshAccountModels(id),
    onSuccess: (_d, id) => void qc.invalidateQueries({ queryKey: qk.accountModels(id) }),
  })
}

export function useRefreshAllAccountsModels() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => refreshAllAccountsModels(),
    onSuccess: () => void qc.invalidateQueries({ queryKey: qk.accounts }),
  })
}

export function useSetAccountOverage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      setAccountOverage(id, enabled),
    onSuccess: (_d, { id }) => {
      void qc.invalidateQueries({ queryKey: qk.accounts })
      void qc.invalidateQueries({ queryKey: qk.accountOverage(id) })
    },
  })
}

export function useConsumeCodexResetCredit() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, creditId }: { id: string; creditId?: string }) =>
      consumeCodexResetCredit(id, creditId),
    onSuccess: (_d, { id }) => {
      void qc.invalidateQueries({ queryKey: qk.accounts })
      void qc.invalidateQueries({ queryKey: qk.accountFull(id) })
    },
  })
}

