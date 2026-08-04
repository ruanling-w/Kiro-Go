// useMediaQuery — subscribe to a CSS media query. SPA-only (no SSR), so the
// initial read can happen synchronously on first client render.
import { useSyncExternalStore } from 'react'

function subscribe(query: string, onChange: () => void) {
  const mql = window.matchMedia(query)
  mql.addEventListener('change', onChange)
  return () => mql.removeEventListener('change', onChange)
}

function getSnapshot(query: string) {
  return window.matchMedia(query).matches
}

/** True when the viewport matches `query` (e.g. `(max-width: 767px)`). */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => subscribe(query, onChange),
    () => getSnapshot(query),
    () => false,
  )
}

/** Tailwind `md` breakpoint is 768px — mobile is everything below. */
export function useIsMobile(): boolean {
  return useMediaQuery('(max-width: 767px)')
}
