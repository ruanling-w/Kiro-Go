// Topbar — theme cycle, language switch, and logout. Sits above the routed page
// content inside AppShell. Theme/lang read from uiStore; logout goes through the
// cookie-session mutation.
import { useTranslation } from 'react-i18next'
import { Moon, Sun, MonitorCog, Languages, LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useUiStore, type ThemePref } from '@/stores/uiStore'
import { useLogout } from '@/hooks/useAuth'
import { LANGS, type Lang } from '@/lib/i18n'

const THEME_ICON: Record<ThemePref, typeof Sun> = {
  system: MonitorCog,
  light: Sun,
  dark: Moon,
}

export function Topbar() {
  const { t } = useTranslation()
  const theme = useUiStore((s) => s.theme)
  const cycleTheme = useUiStore((s) => s.cycleTheme)
  const lang = useUiStore((s) => s.lang)
  const setLang = useUiStore((s) => s.setLang)
  const logout = useLogout()
  const ThemeIcon = THEME_ICON[theme]

  return (
    <header className="flex h-16 shrink-0 items-center justify-end gap-1 border-b bg-background px-4">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" aria-label={t('lang.label')}>
            <Languages className="size-5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {LANGS.map((l) => (
            <DropdownMenuItem
              key={l}
              onClick={() => setLang(l as Lang)}
              className={l === lang ? 'font-semibold' : ''}
            >
              {t(`lang.${l}`)}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <Button variant="ghost" size="icon" aria-label={t('theme.toggle')} onClick={cycleTheme}>
        <ThemeIcon className="size-5" />
      </Button>

      <Button
        variant="ghost"
        size="icon"
        aria-label={t('common.logout')}
        onClick={() => logout.mutate()}
        disabled={logout.isPending}
      >
        <LogOut className="size-5" />
      </Button>
    </header>
  )
}
