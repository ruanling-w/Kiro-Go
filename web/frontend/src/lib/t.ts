// Translation helper that bridges the legacy positional-placeholder locale format
// ({0}, {1}, …) to i18next. Call sites use either:
//   t('some.key')                    → plain lookup
//   t('some.key', 'foo', 42)         → fills {0}='foo', {1}=42
//
// The variadic args are turned into an index-keyed object ({ 0: 'foo', 1: 42 })
// which i18next interpolates via the {0}/{1} prefix/suffix configured in i18n.ts.
//
// For use inside React components prefer the useTranslation() hook so the tree
// re-renders on language change; this standalone helper is for non-component code
// (services, stores, one-off formatting) and reads the current i18n instance.
import i18n from './i18n'
import type { TFunction } from 'i18next'

/** Turn positional args into the index-keyed object i18next interpolates. */
function toValues(args: (string | number)[]): Record<string, string | number> {
  const values: Record<string, string | number> = {}
  args.forEach((a, i) => {
    values[i] = a
  })
  return values
}

export function t(key: string, ...args: (string | number)[]): string {
  if (args.length === 0) return i18n.t(key)
  return i18n.t(key, toValues(args))
}

/**
 * Positional-arg wrapper around a react-i18next TFunction (from useTranslation),
 * so components can write tp(t, 'key', a, b) instead of t('key', {0:a, 1:b}) and
 * still re-render on language change. Use this inside components; use `t` above
 * for non-component code.
 */
export function tp(tf: TFunction, key: string, ...args: (string | number)[]): string {
  if (args.length === 0) return tf(key)
  return tf(key, toValues(args))
}
