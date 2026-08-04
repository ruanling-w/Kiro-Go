// Sidebar — the primary navigation. Desktop: fixed left rail (collapsible to
// an icon strip via uiStore.sidebarCollapsed). Mobile: rendered inside a Sheet
// from AppShell (same NavLinks; layoutId active pill only on desktop so Motion
// doesn't fight the drawer mount/unmount).
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { motion } from 'motion/react'
import { NAV_ITEMS } from '@/config/nav'
import { prefersReducedMotion, EASE_OUT } from '@/lib/motion'
import { cn } from '@/lib/utils'
import { useUiStore } from '@/stores/uiStore'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function SidebarBrand({ collapsed = false }: { collapsed?: boolean }) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        'flex h-16 items-center border-b',
        collapsed ? 'justify-center px-2' : 'gap-2.5 px-5',
      )}
    >
      <img
        src="/admin/logo.png"
        alt=""
        className={cn(
          'shrink-0 object-contain',
          collapsed ? 'size-8' : 'size-10 md:size-12',
        )}
        onError={(e) => (e.currentTarget.style.display = 'none')}
      />
      {!collapsed && (
        <span className="truncate font-semibold tracking-tight">{t('app.title')}</span>
      )}
    </div>
  )
}

interface SidebarNavProps {
  /** Called after a nav link is activated (close mobile drawer). */
  onNavigate?: () => void
  /** When false, skip shared layoutId so desktop/mobile don't share Motion state. */
  animated?: boolean
  /** Icon-only rail (desktop collapsed). */
  collapsed?: boolean
  className?: string
}

export function SidebarNav({
  onNavigate,
  animated = true,
  collapsed = false,
  className,
}: SidebarNavProps) {
  const { t } = useTranslation()
  const reduce = prefersReducedMotion()

  const links = NAV_ITEMS.map((item) => {
    const Icon = item.icon
    const label = t(item.labelKey)
    const link = (
      <NavLink
        key={item.path}
        to={item.path}
        end={item.path === '/'}
        onClick={onNavigate}
        title={collapsed ? label : undefined}
        aria-label={collapsed ? label : undefined}
        className={({ isActive }) =>
          cn(
            'group/nav relative flex items-center text-sm font-medium',
            'transition-[color,background-color,transform] duration-200 active:scale-[0.97]',
            collapsed
              ? 'mx-auto size-11 justify-center rounded-xl'
              : 'gap-3 rounded-lg px-3 py-2',
            isActive
              ? 'text-primary'
              : 'text-sidebar-foreground/65 hover:bg-sidebar-accent/35 hover:text-sidebar-foreground',
          )
        }
      >
        {({ isActive }) => (
          <>
            {/* Active affordance = borderless tinted pill + a flush accent bar.
                Rendered as Motion layers on desktop so the pill glides between
                items; plain spans on mobile (no shared layout across mounts). */}
            {isActive &&
              (animated ? (
                <>
                  <motion.span
                    layoutId="nav-active"
                    className={cn(
                      'absolute inset-0 bg-primary/10 dark:bg-primary/15',
                      collapsed ? 'rounded-xl' : 'rounded-lg',
                    )}
                    transition={reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }}
                  />
                  <motion.span
                    layoutId="nav-active-bar"
                    className={cn(
                      'absolute rounded-full bg-primary',
                      collapsed
                        ? 'inset-x-0 -bottom-0.5 mx-auto h-1 w-5'
                        : 'top-1/2 left-0 h-5 w-1 -translate-y-1/2',
                    )}
                    transition={reduce ? { duration: 0 } : { duration: 0.3, ease: EASE_OUT }}
                  />
                </>
              ) : (
                <>
                  <span
                    className={cn(
                      'absolute inset-0 bg-primary/10 dark:bg-primary/15',
                      collapsed ? 'rounded-xl' : 'rounded-lg',
                    )}
                  />
                  <span
                    className={cn(
                      'absolute rounded-full bg-primary',
                      collapsed
                        ? 'inset-x-0 -bottom-0.5 mx-auto h-1 w-5'
                        : 'top-1/2 left-0 h-5 w-1 -translate-y-1/2',
                    )}
                  />
                </>
              ))}
            <Icon
              className={cn(
                'relative shrink-0 transition-transform duration-200',
                collapsed ? 'size-5' : 'size-4.5',
                !isActive && 'group-hover/nav:scale-110',
              )}
              strokeWidth={isActive ? 2.25 : 2}
            />
            {!collapsed && <span className="relative truncate">{label}</span>}
          </>
        )}
      </NavLink>
    )

    if (!collapsed) return link

    return (
      <Tooltip key={item.path}>
        <TooltipTrigger asChild>{link}</TooltipTrigger>
        <TooltipContent side="right" sideOffset={8}>
          {label}
        </TooltipContent>
      </Tooltip>
    )
  })

  return (
    <TooltipProvider delayDuration={200}>
      <nav
        className={cn(
          'flex-1',
          collapsed ? 'flex flex-col items-center gap-2 px-2 py-3' : 'space-y-1 p-3',
          className,
        )}
      >
        {links}
      </nav>
    </TooltipProvider>
  )
}

export function Sidebar() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed)

  return (
    <aside
      className={cn(
        'hidden shrink-0 flex-col border-r bg-sidebar transition-[width] duration-200 ease-out md:flex',
        collapsed ? 'w-16' : 'w-60',
      )}
    >
      <SidebarBrand collapsed={collapsed} />
      <SidebarNav animated collapsed={collapsed} />
    </aside>
  )
}
