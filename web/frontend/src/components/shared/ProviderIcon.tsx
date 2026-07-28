// ProviderIcon — the bucket's lucide icon in its brand tint. Used on the
// Providers landing cards and account cards.
import { providerMeta, ALL_PROVIDER } from '@/config/providers'
import { cn } from '@/lib/utils'

interface ProviderIconProps {
  provider: string
  className?: string
}

export function ProviderIcon({ provider, className }: ProviderIconProps) {
  const meta = providerMeta(provider)
  const Icon = meta?.icon ?? ALL_PROVIDER.icon
  const color = meta?.color ?? ALL_PROVIDER.color
  return <Icon className={cn(color, className)} />
}
