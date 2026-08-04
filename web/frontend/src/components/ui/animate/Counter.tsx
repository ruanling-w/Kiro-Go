// Counter — count-up animation for KPI/stat numerals. Springs from the previous
// value to the next and formats each frame via `format`. Respects reduced-motion
// by snapping straight to the target. Copy-source primitive (AGENT.md §7).
import { useEffect, useRef, useState } from 'react'
import { animate } from 'motion/react'
import { prefersReducedMotion, EASE_OUT } from '@/lib/motion'

interface CounterProps {
  value: number
  format?: (n: number) => string
  className?: string
  duration?: number
}

export function Counter({
  value,
  format = (n) => Math.round(n).toLocaleString(),
  className,
  duration = 0.9,
}: CounterProps) {
  const [display, setDisplay] = useState(value)
  const fromRef = useRef(value)
  // KPI counts are integers — round every frame so formatCompact/etc. never
  // flash "241.9998" mid-tween when status/SSE updates arrive.
  const snap = Number.isFinite(value) && Number.isInteger(value)

  useEffect(() => {
    const from = fromRef.current
    fromRef.current = value
    if (from === value) return
    if (prefersReducedMotion()) {
      setDisplay(value)
      return
    }
    const controls = animate(from, value, {
      duration,
      ease: EASE_OUT,
      onUpdate: (v) => setDisplay(snap ? Math.round(v) : v),
    })
    return () => controls.stop()
  }, [value, duration, snap])

  return <span className={className}>{format(display)}</span>
}
