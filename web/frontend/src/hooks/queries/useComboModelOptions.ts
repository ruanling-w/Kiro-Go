// Aggregates the models of every logged-in provider into grouped options for the
// Combo model pickers. A provider bucket is "logged in" when at least one enabled
// account maps to it (see bucketOf). Each active bucket is fetched via the same
// /providers/{provider}/models endpoint the Providers page uses.
import { useMemo } from 'react'
import type { ProviderKey } from '@/types/account'
import type { ProviderModel } from '@/types/provider'
import { bucketOf, providerMeta } from '@/config/providers'
import { useAccounts } from './useAccounts'
import { useProviderModels } from './useProviderModels'

export interface ComboModelGroup {
  key: ProviderKey
  labelKey: string
  models: ProviderModel[]
}

export function useComboModelOptions() {
  const { data: accounts } = useAccounts()

  const activeKeys = useMemo(() => {
    const keys = new Set<ProviderKey>()
    for (const a of accounts ?? []) {
      if (a.enabled) keys.add(bucketOf(a.provider))
    }
    return keys
  }, [accounts])

  // Fixed hook order: one query per known provider, gated by whether an account exists.
  const kiro = useProviderModels('kiro', activeKeys.has('kiro'))
  const antigravity = useProviderModels('antigravity', activeKeys.has('antigravity'))
  const grok = useProviderModels('grok', activeKeys.has('grok'))
  const codex = useProviderModels('codex', activeKeys.has('codex'))
  const remotekiro = useProviderModels('remotekiro', activeKeys.has('remotekiro'))
  const voyage = useProviderModels('voyage', activeKeys.has('voyage'))

  const byKey: Record<ProviderKey, ReturnType<typeof useProviderModels>> = {
    kiro,
    antigravity,
    grok,
    codex,
    remotekiro,
    voyage,
  }
  const order: ProviderKey[] = ['kiro', 'antigravity', 'grok', 'codex', 'remotekiro', 'voyage']

  const groups = useMemo<ComboModelGroup[]>(
    () =>
      order
        .filter((k) => activeKeys.has(k))
        .map((k) => ({
          key: k,
          labelKey: providerMeta(k)?.labelKey ?? k,
          models: byKey[k].data?.models ?? [],
        }))
        .filter((g) => g.models.length > 0),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [activeKeys, kiro.data, antigravity.data, grok.data, codex.data, remotekiro.data, voyage.data],
  )

  const isLoading = order.some((k) => activeKeys.has(k) && byKey[k].isPending)
  const hasActiveProviders = activeKeys.size > 0

  return { groups, isLoading, hasActiveProviders }
}
