// Account read hooks. The list feeds every account/dashboard view; full/models/
// overage are fetched on demand from dialogs.
import { useQuery } from '@tanstack/react-query'
import { qk } from '@/config/queryKeys'
import {
  listAccounts,
  getAccountFull,
  getAccountOverage,
} from '@/services/accounts.service'

export function useAccounts() {
  return useQuery({
    queryKey: qk.accounts,
    queryFn: listAccounts,
    refetchInterval: 60_000,
  })
}

export function useAccountFull(id: string | null) {
  return useQuery({
    queryKey: id ? qk.accountFull(id) : qk.accountFull('none'),
    queryFn: () => getAccountFull(id as string),
    enabled: !!id,
  })
}

export function useAccountOverage(id: string | null, enabled = true) {
  return useQuery({
    queryKey: id ? qk.accountOverage(id) : qk.accountOverage('none'),
    queryFn: () => getAccountOverage(id as string),
    enabled: !!id && enabled,
  })
}
