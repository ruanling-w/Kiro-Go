// ProviderIcon — the bucket's brand logo (from public/) when one exists, else
// its lucide icon in the brand tint. Used on the Providers landing cards and
// account cards. On a logo load failure we swap back to the lucide icon.
import { useState } from 'react'
import { providerMeta, ALL_PROVIDER } from '@/config/providers'
import { cn } from '@/lib/utils'

interface ProviderIconProps {
  provider: string
  className?: string
}

export function ProviderIcon({ provider, className }: ProviderIconProps) {
  const meta = providerMeta(provider)
  const [broken, setBroken] = useState(false)

  if (meta?.logo && !broken) {
    return (
      <img
        src={meta.logo}
        alt=""
        aria-hidden
        className={cn('size-5 shrink-0 object-contain', className)}
        onError={() => setBroken(true)}
      />
    )
  }

  const Icon = meta?.icon ?? ALL_PROVIDER.icon
  const color = meta?.color ?? ALL_PROVIDER.color
  return <Icon className={cn(color, className)} />
}
