// Shared motion tokens + reduced-motion helper. Centralizes easing/duration and
// reusable variants so components don't scatter magic numbers. Every animation
// in the app should pull from here and branch on prefersReducedMotion().
import type { Transition, Variants } from 'motion/react'

export function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

// Easing tuned for UI: a soft ease-out for enters, snappy for micro-interactions.
export const EASE_OUT = [0.16, 1, 0.3, 1] as const
export const EASE_SNAP = [0.4, 0, 0.2, 1] as const

export const DURATION = {
  micro: 0.15,
  fast: 0.22,
  base: 0.3,
} as const

export const pageTransition: Transition = {
  duration: DURATION.base,
  ease: EASE_OUT,
}

// Fade + slight rise for page/content reveals.
export const fadeRise: Variants = {
  hidden: { opacity: 0, y: 8 },
  visible: { opacity: 1, y: 0 },
}

// Stagger container for KPI rows / card grids.
export const staggerContainer: Variants = {
  hidden: {},
  visible: {
    transition: { staggerChildren: 0.06, delayChildren: 0.02 },
  },
}
