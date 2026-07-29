// Provider bucket metadata: the 5 real provider buckets + the synthetic "all"
// bucket used by the Providers landing / Accounts filter. `labelKey`/`descKey`
// are i18n keys; `color` is a Tailwind text class for the ProviderIcon tint.
import { Boxes, Sparkles, Bot, Terminal, Cloud, Layers, type LucideIcon } from 'lucide-react'
import type { ProviderKey } from '@/types/account'

export interface ProviderMeta {
  key: ProviderKey
  labelKey: string
  descKey: string
  icon: LucideIcon
  color: string
  /** Brand logo under public/ (served at /admin/). Falls back to `icon` when absent. */
  logo?: string
  /** Which raw account.provider values map into this bucket. */
  match: (provider: string) => boolean
}

export const PROVIDERS: ProviderMeta[] = [
  {
    key: 'kiro',
    labelKey: 'provider.kiro',
    descKey: 'providerDesc.kiro',
    icon: Boxes,
    color: 'text-violet-500',
    logo: '/admin/kiro.svg',
    match: (p) => p === 'kiro' || p === '' || p === 'builderid' || p === 'iam-sso' || p === 'kiro-sso',
  },
  {
    key: 'antigravity',
    labelKey: 'provider.antigravity',
    descKey: 'providerDesc.antigravity',
    icon: Sparkles,
    color: 'text-sky-500',
    logo: '/admin/antigravity-color.svg',
    match: (p) => p === 'antigravity',
  },
  {
    key: 'grok',
    labelKey: 'provider.grok',
    descKey: 'providerDesc.grok',
    icon: Bot,
    color: 'text-orange-500',
    logo: '/admin/grok.webp',
    match: (p) => p === 'grok',
  },
  {
    key: 'codex',
    labelKey: 'provider.codex',
    descKey: 'providerDesc.codex',
    icon: Terminal,
    color: 'text-emerald-500',
    logo: '/admin/codex-color.svg',
    match: (p) => p === 'codex',
  },
  {
    key: 'remotekiro',
    labelKey: 'provider.remotekiro',
    descKey: 'providerDesc.remotekiro',
    icon: Cloud,
    color: 'text-teal-500',
    match: (p) => p === 'remotekiro' || p === 'remote-kiro',
  },
]

export const ALL_PROVIDER = {
  icon: Layers,
  color: 'text-foreground',
}

export function providerMeta(key: string): ProviderMeta | undefined {
  return PROVIDERS.find((p) => p.key === key)
}

/** Bucket an account's raw provider string into one of the known ProviderKeys. */
export function bucketOf(provider: string): ProviderKey {
  const p = (provider || '').toLowerCase()
  return PROVIDERS.find((meta) => meta.match(p))?.key ?? 'kiro'
}
