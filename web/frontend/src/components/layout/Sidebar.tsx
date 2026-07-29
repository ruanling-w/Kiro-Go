// Sidebar — the 6-page primary navigation. Uses NavLink so the active route is
// highlighted automatically; `end` on the dashboard link stops it matching every
// nested path. The active pill slides between items via a shared layoutId.
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { motion } from 'motion/react'
import { NAV_ITEMS } from '@/config/nav'
import { prefersReducedMotion, EASE_OUT } from '@/lib/motion'
import { cn } from '@/lib/utils'

export function Sidebar() {
  const { t } = useTranslation()
  const reduce = prefersReducedMotion()
  return (
    <aside className="flex w-60 shrink-0 flex-col border-r bg-sidebar">
      <div className="flex h-16 items-center gap-2.5 border-b px-5">
        <img
          src="/admin/logo.png"
          alt=""
          className="size-12 shrink-0 object-contain"
          onError={(e) => (e.currentTarget.style.display = 'none')}
        />
        <span className="font-semibold tracking-tight">{t('app.title')}</span>
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {NAV_ITEMS.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === '/'}
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
                  {isActive && (
                    <motion.span
                      layoutId="nav-active"
                      className="absolute inset-0 rounded-lg bg-sidebar-accent ring-1 ring-primary/25"
                      transition={
                        reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }
                      }
                    />
                  )}
                  {isActive && (
                    <motion.span
                      layoutId="nav-active-bar"
                      className="absolute top-1/2 left-0 h-5 w-1 -translate-y-1/2 rounded-r-full bg-primary"
                      transition={
                        reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }
                      }
                    />
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
    </aside>
  )
}
