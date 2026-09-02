import { useMemo } from 'react'
import type { ProviderKey } from '@/types/account'
import type { ProviderModel } from '@/types/provider'
import { bucketOf, providerMeta } from '@/config/providers'
import { displayEmail } from '@/lib/mask'
import { useAccounts } from './useAccounts'
import { useProviderModels } from './useProviderModels'

export interface ComboModelGroup {
  key: string
  provider: string
  targetProvider: string
  label?: string
  labelKey?: string
  subLabel?: string
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
  const standardProviders: ProviderKey[] = ['kiro', 'antigravity', 'grok', 'codex', 'voyage']

  const groups = useMemo<ComboModelGroup[]>(() => {
    const result: ComboModelGroup[] = []
    const enabledAccounts = (accounts ?? []).filter((a) => a.enabled)

    // 1. Standard provider groups
    for (const k of standardProviders) {
      if (!activeKeys.has(k)) continue
      const models = byKey[k].data?.models ?? []
      if (models.length > 0) {
        result.push({
          key: k,
          provider: k,
          targetProvider: k,
          labelKey: providerMeta(k)?.labelKey ?? k,
          models,
        })
      }
    }

    // 2. Separate groups per Remote Kiro / Custom API account
    const remoteAccounts = enabledAccounts.filter((a) => bucketOf(a.provider) === 'remotekiro')
    for (const acc of remoteAccounts) {
      const customModels = acc.customModels ?? []
      let models: ProviderModel[] = []
      if (customModels.length > 0) {
        models = customModels.map((m) => ({
          id: m,
          name: m,
          description: '',
          supports_image: false,
        }))
      } else {
        models = remotekiro.data?.models ?? []
      }

      if (models.length > 0) {
        const nickname = acc.nickname?.trim()
        const target = nickname || acc.id
        const email = acc.email ? displayEmail(acc.email, acc.id, false) : ''
        const label = nickname ? `${nickname}${email ? ` (${email})` : ''}` : email || acc.id.slice(0, 8)

        result.push({
          key: acc.id,
          provider: 'remotekiro',
          targetProvider: target,
          label,
          models,
        })
      }
    }

    return result
  }, [accounts, activeKeys, kiro.data, antigravity.data, grok.data, codex.data, remotekiro.data, voyage.data])

  const allProviders: ProviderKey[] = ['kiro', 'antigravity', 'grok', 'codex', 'remotekiro', 'voyage']
  const isLoading = allProviders.some((k) => activeKeys.has(k) && byKey[k].isPending)
  const hasActiveProviders = activeKeys.size > 0

  return { groups, isLoading, hasActiveProviders }
}

