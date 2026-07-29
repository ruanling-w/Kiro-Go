// MotionPage — page-transition wrapper: fade + slight rise on route change.
// Keyed by the caller (route path) so each page mounts fresh and animates in.
// Reduced-motion collapses the transition to an instant show. Copy-source
// primitive (AGENT.md §7).
import type { ReactNode } from 'react'
import { motion } from 'motion/react'
import { prefersReducedMotion, pageTransition, fadeRise } from '@/lib/motion'

export function MotionPage({ children }: { children: ReactNode }) {
  if (prefersReducedMotion()) return <>{children}</>
  return (
    <motion.div
      variants={fadeRise}
      initial="hidden"
      animate="visible"
      transition={pageTransition}
      className="h-full"
    >
      {children}
    </motion.div>
  )
}
