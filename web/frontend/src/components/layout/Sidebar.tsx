// Sidebar — the 6-page primary navigation. Uses NavLink so the active route is
// highlighted automatically; `end` on the dashboard link stops it matching every
// nested path.
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { NAV_ITEMS } from '@/config/nav'
import { cn } from '@/lib/utils'

export function Sidebar() {
  const { t } = useTranslation()
  return (
    <aside className="flex w-60 shrink-0 flex-col border-r bg-sidebar">
      <div className="flex h-16 items-center gap-2.5 border-b px-5">
        <img src="/admin/logo.png" alt="" className="size-10 rounded-lg object-contain" onError={(e) => (e.currentTarget.style.display = 'none')} />
        <span className="font-semibold">{t('app.title')}</span>
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
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                    : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
                )
              }
            >
              <Icon className="size-4.5 shrink-0" />
              <span>{t(item.labelKey)}</span>
            </NavLink>
          )
        })}
      </nav>
    </aside>
  )
}
