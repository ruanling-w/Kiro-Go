// Sidebar navigation config — the 6 admin pages (Login is outside the shell).
//
// `labelKey` is an i18n key resolved at render time (not the label itself) so
// the nav re-translates on language change. `path` is the route; `icon` is a
// lucide component.
import {
  LayoutDashboard,
  Boxes,
  Users,
  KeyRound,
  Settings2,
  ScrollText,
  type LucideIcon,
} from 'lucide-react'

export interface NavItem {
  path: string
  labelKey: string
  icon: LucideIcon
}

export const NAV_ITEMS: NavItem[] = [
  { path: '/', labelKey: 'nav.overview', icon: LayoutDashboard },
  { path: '/providers', labelKey: 'nav.providers', icon: Boxes },
  { path: '/accounts', labelKey: 'nav.allAccounts', icon: Users },
  { path: '/api-keys', labelKey: 'nav.apikeys', icon: KeyRound },
  { path: '/settings', labelKey: 'nav.system', icon: Settings2 },
  { path: '/logs', labelKey: 'logs.title', icon: ScrollText },
]
