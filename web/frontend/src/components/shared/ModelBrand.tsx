// ModelBrand — one palette per upstream provider bucket, shared by the request
// log table and the Combo route chain so the same model id looks identical
// wherever it appears (violet Kiro/Claude, zinc Grok, sky Antigravity, …).
import { useTranslation } from 'react-i18next'
import { ProviderIcon } from '@/components/shared/ProviderIcon'
import { bucketOf, providerMeta } from '@/config/providers'
import { cn } from '@/lib/utils'

const PROVIDER_STYLE: Record<
  string,
  { logo?: string; chip: string; text: string }
> = {
  kiro: {
    logo: '/admin/kiro.svg',
    chip: 'border-violet-500/25 bg-violet-500/10',
    text: 'text-violet-700 dark:text-violet-300',
  },
  antigravity: {
    logo: '/admin/antigravity-color.svg',
    chip: 'border-sky-500/25 bg-sky-500/10',
    text: 'text-sky-700 dark:text-sky-300',
  },
  grok: {
    logo: '/admin/grok.webp',
    chip: 'border-zinc-500/30 bg-zinc-500/10 dark:border-zinc-400/25 dark:bg-zinc-400/10',
    text: 'text-zinc-700 dark:text-zinc-200',
  },
  codex: {
    logo: '/admin/codex-color.svg',
    chip: 'border-emerald-500/25 bg-emerald-500/10',
    text: 'text-emerald-700 dark:text-emerald-300',
  },
  remotekiro: {
    logo: '/admin/kiro.svg',
    chip: 'border-teal-500/25 bg-teal-500/10',
    text: 'text-teal-700 dark:text-teal-300',
  },
}

/** Guess bucket from the model id (and provider hint when present). */
export function guessBucket(model?: string, provider?: string): string | null {
  const s = `${provider || ''} ${model || ''}`.toLowerCase()
  if (!s.trim()) return null
  if (s.includes('grok') || s.includes('xai')) return 'grok'
  if (s.includes('gemini') || s.includes('antigravity')) return 'antigravity'
  if (
    s.includes('gpt') ||
    s.includes('codex') ||
    /\bo[1-4]\b/.test(s) ||
    s.includes('openai')
  ) {
    return 'codex'
  }
  if (s.includes('kiro') || s.includes('codewhisperer') || s.includes('amazonq')) {
    return 'kiro'
  }
  // Claude/sonnet models often ride on Kiro upstream when provider missing.
  if (s.includes('claude') || s.includes('sonnet') || s.includes('opus') || s.includes('haiku')) {
    return 'kiro'
  }
  return null
}

export type Brand = {
  key: string
  label: string
  logo?: string
  chip: string
  text: string
}

/**
 * Resolve the brand for a row/chip. `provider` wins when the backend recorded
 * one; otherwise the model id is sniffed (legacy log rows, Combo model lists
 * which carry no provider at all).
 */
export function brandFor(
  provider: string | undefined,
  model: string | undefined,
  t: (k: string) => string,
): Brand | null {
  const key = provider?.trim() ? bucketOf(provider) : guessBucket(model, provider)
  if (!key) return null
  const style = PROVIDER_STYLE[key]
  if (!style) return null
  const meta = providerMeta(key)
  return {
    key,
    label: meta ? t(meta.labelKey) : key,
    logo: meta?.logo ?? style.logo,
    chip: style.chip,
    text: style.text,
  }
}

export function BrandChip({
  brand,
  text,
  title,
  className,
}: {
  brand: Brand
  text: string
  title?: string
  className?: string
}) {
  return (
    <span
      title={title || text}
      className={cn(
        'inline-flex max-w-full items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-semibold',
        brand.chip,
        brand.text,
        className,
      )}
    >
      {brand.logo ? (
        <img
          src={brand.logo}
          alt=""
          aria-hidden
          className="size-3.5 shrink-0 object-contain"
          onError={(e) => {
            e.currentTarget.style.display = 'none'
          }}
        />
      ) : (
        <ProviderIcon provider={brand.key} className="size-3.5 shrink-0" />
      )}
      <span className="truncate">{text}</span>
    </span>
  )
}

/**
 * Just the brand mark for a model id — for tight rows where a full chip would
 * not fit (the ordered picker, free-text inputs). Renders nothing recognizable
 * for unknown ids, so callers keep their own placeholder.
 */
export function ModelIcon({
  model,
  provider,
  className,
}: {
  model?: string
  provider?: string
  className?: string
}) {
  const { t } = useTranslation()
  const brand = brandFor(provider, model, t)
  if (!brand) return null
  return (
    <ProviderIcon
      provider={brand.key}
      className={cn('size-4 shrink-0', className)}
    />
  )
}

/**
 * A model id rendered in its provider colors. Falls back to a neutral mono chip
 * when the id matches no known bucket (free-text combo entries, typos).
 */
export function ModelChip({
  model,
  provider,
  className,
}: {
  model: string
  provider?: string
  className?: string
}) {
  const { t } = useTranslation()
  const brand = brandFor(provider, model, t)
  if (!brand) {
    return (
      <span
        title={model}
        className={cn(
          'inline-flex max-w-full items-center rounded-full border bg-background px-2 py-0.5 text-[11px] font-semibold text-muted-foreground',
          className,
        )}
      >
        <span className="truncate">{model}</span>
      </span>
    )
  }
  return <BrandChip brand={brand} text={model} title={model} className={className} />
}
