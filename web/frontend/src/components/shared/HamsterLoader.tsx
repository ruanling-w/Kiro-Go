// HamsterLoader — the single shared loading indicator for the whole app.
//
// Ported verbatim (markup + CSS) from the legacy admin's loaderHtml() so the
// beloved hamster-in-a-wheel animation carries over. Use it everywhere data is
// loading: full-page route fallback, inside a card/table while a query is
// pending, inside a button while a mutation runs, or inside a dialog.
//
// Do NOT introduce other spinners — this is the app-wide standard. For layout
// placeholders (skeletons) use the shimmer components; for "working" always use
// this.
import './HamsterLoader.css'
import { cn } from '@/lib/utils'

interface HamsterLoaderProps {
  /** Optional label rendered under the wheel. */
  label?: string
  /** 'sm' shrinks the wheel for inline/button use; 'md' (default) is full size. */
  size?: 'sm' | 'md'
  className?: string
}

/** The wheel + hamster on its own, no padding — for tight inline spots. */
export function HamsterWheel({ size = 'md' }: { size?: 'sm' | 'md' }) {
  return (
    <div
      className={cn('wheel-and-hamster', size === 'sm' && 'wheel-and-hamster--sm')}
      role="img"
      aria-label="Loading"
    >
      <div className="wheel" />
      <div className="hamster">
        <div className="hamster__body">
          <div className="hamster__head">
            <div className="hamster__ear" />
            <div className="hamster__eye" />
            <div className="hamster__nose" />
          </div>
          <div className="hamster__limb hamster__limb--fr" />
          <div className="hamster__limb hamster__limb--fl" />
          <div className="hamster__limb hamster__limb--br" />
          <div className="hamster__limb hamster__limb--bl" />
          <div className="hamster__tail" />
        </div>
      </div>
      <div className="spoke" />
    </div>
  )
}

export function HamsterLoader({ label, size = 'md', className }: HamsterLoaderProps) {
  return (
    <div className={cn('hamster-loader', className)} role="status" aria-live="polite">
      <HamsterWheel size={size} />
      {label ? <span className="hamster-loader-label">{label}</span> : null}
    </div>
  )
}

/** Full-viewport centered loader — use as a Suspense/route fallback. */
export function FullPageLoader({ label }: { label?: string }) {
  return (
    <div className="flex min-h-dvh items-center justify-center">
      <HamsterLoader label={label} />
    </div>
  )
}
