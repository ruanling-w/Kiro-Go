// Topbar — live status dot, theme cycle, language switch, and logout. Sits above
// the routed page content inside AppShell. Theme/lang read from uiStore; logout
// goes through the cookie-session mutation. The live dot reflects the real
// status poll (useStatus, 10s): teal pulse when data is flowing, muted on error.
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
import { useStatus } from '@/hooks/queries/useStatus'
import { useLogout } from '@/hooks/useAuth'
import { LANGS, type Lang } from '@/lib/i18n'
import { cn } from '@/lib/utils'

const THEME_ICON: Record<ThemePref, typeof Sun> = {
  system: MonitorCog,
  light: Sun,
  dark: Moon,
}

function LiveDot() {
  const { t } = useTranslation()
  const status = useStatus()
  const live = status.isSuccess && !status.isError
  return (
    <div className="mr-1 flex items-center gap-2 rounded-full border border-border bg-muted/40 px-2.5 py-1">
      <span className="relative flex size-2">
        {live && (
          <span className="absolute inline-flex size-full animate-ping rounded-full bg-primary/70" />
        )}
        <span
          className={cn(
            'relative inline-flex size-2 rounded-full',
            live ? 'bg-primary' : 'bg-muted-foreground/50',
          )}
        />
      </span>
      <span className="text-xs font-medium text-muted-foreground">{t('status.live')}</span>
    </div>
  )
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
      <LiveDot />
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
