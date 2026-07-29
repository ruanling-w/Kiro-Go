// FadeIn / Stagger — content reveal primitives. `Stagger` is a container that
// releases its `FadeIn` children in sequence (KPI rows, card grids); `FadeIn`
// can also stand alone for a single reveal. Reduced-motion renders children
// statically. Copy-source primitive (AGENT.md §7).
import type { ReactNode } from 'react'
import { motion } from 'motion/react'
import {
  prefersReducedMotion,
  pageTransition,
  fadeRise,
  staggerContainer,
} from '@/lib/motion'
import { cn } from '@/lib/utils'

interface StaggerProps {
  children: ReactNode
  className?: string
}

export function Stagger({ children, className }: StaggerProps) {
  if (prefersReducedMotion()) return <div className={className}>{children}</div>
  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className={className}
    >
      {children}
    </motion.div>
  )
}

interface FadeInProps {
  children: ReactNode
  className?: string
  /** Delay in seconds when used standalone (outside a Stagger container). */
  delay?: number
}

export function FadeIn({ children, className, delay }: FadeInProps) {
  if (prefersReducedMotion()) return <div className={className}>{children}</div>
  return (
    <motion.div
      variants={fadeRise}
      initial="hidden"
      animate="visible"
      transition={{ ...pageTransition, delay }}
      className={cn(className)}
    >
      {children}
    </motion.div>
  )
}

// StaggerItem — a reveal child that INHERITS initial/animate from a parent
// <Stagger>. It only declares variants (no initial/animate of its own) so the
// container's orchestration drives the sequence; setting them here would make
// the child animate immediately and defeat the stagger.
export function StaggerItem({ children, className }: StaggerProps) {
  if (prefersReducedMotion()) return <div className={className}>{children}</div>
  return (
    <motion.div variants={fadeRise} transition={pageTransition} className={className}>
      {children}
    </motion.div>
  )
}
