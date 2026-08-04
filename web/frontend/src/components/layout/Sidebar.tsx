// Sidebar — the primary navigation. Desktop: fixed left rail. Mobile: rendered
// inside a Sheet from AppShell (same NavLinks; layoutId active pill only on
// desktop so Motion doesn't fight the drawer mount/unmount).
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { motion } from 'motion/react'
import { NAV_ITEMS } from '@/config/nav'
import { prefersReducedMotion, EASE_OUT } from '@/lib/motion'
import { cn } from '@/lib/utils'

export function SidebarBrand() {
  const { t } = useTranslation()
  return (
    <div className="flex h-16 items-center gap-2.5 border-b px-5">
      <img
        src="/admin/logo.png"
        alt=""
        className="size-12 shrink-0 object-contain"
        onError={(e) => (e.currentTarget.style.display = 'none')}
      />
      <span className="font-semibold tracking-tight">{t('app.title')}</span>
    </div>
  )
}

interface SidebarNavProps {
  /** Called after a nav link is activated (close mobile drawer). */
  onNavigate?: () => void
  /** When false, skip shared layoutId so desktop/mobile don't share Motion state. */
  animated?: boolean
  className?: string
}

export function SidebarNav({ onNavigate, animated = true, className }: SidebarNavProps) {
  const { t } = useTranslation()
  const reduce = prefersReducedMotion()

  return (
    <nav className={cn('flex-1 space-y-1 p-3', className)}>
      {NAV_ITEMS.map((item) => {
        const Icon = item.icon
        return (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === '/'}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                'group/nav relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/40 hover:text-sidebar-foreground',
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive && animated && (
                  <motion.span
                    layoutId="nav-active"
                    className="absolute inset-0 rounded-lg bg-sidebar-accent ring-1 ring-primary/25"
                    transition={
                      reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }
                    }
                  />
                )}
                {isActive && !animated && (
                  <span className="absolute inset-0 rounded-lg bg-sidebar-accent ring-1 ring-primary/25" />
                )}
                {isActive && animated && (
                  <motion.span
                    layoutId="nav-active-bar"
                    className="absolute top-1/2 left-0 h-5 w-1 -translate-y-1/2 rounded-r-full bg-primary"
                    transition={
                      reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }
                    }
                  />
                )}
                {isActive && !animated && (
                  <span className="absolute top-1/2 left-0 h-5 w-1 -translate-y-1/2 rounded-r-full bg-primary" />
                )}
                <Icon
                  className={cn(
                    'relative size-4.5 shrink-0 transition-colors',
                    isActive && 'text-primary',
                  )}
                />
                <span className="relative">{t(item.labelKey)}</span>
              </>
            )}
          </NavLink>
        )
      })}
    </nav>
  )
}

export function Sidebar() {
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r bg-sidebar md:flex">
      <SidebarBrand />
      <SidebarNav animated />
    </aside>
  )
}
