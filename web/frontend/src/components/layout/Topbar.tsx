// Topbar — live status dot, theme cycle, language switch, and logout. On mobile
// also hosts the hamburger that opens the nav Sheet (wired by AppShell).
import { useTranslation } from 'react-i18next'
import { Menu, Moon, Sun, MonitorCog, Languages, LogOut } from 'lucide-react'
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
      <span className="hidden text-xs font-medium text-muted-foreground sm:inline">
        {t('status.live')}
      </span>
    </div>
  )
}

interface TopbarProps {
  onOpenNav?: () => void
}

export function Topbar({ onOpenNav }: TopbarProps) {
  const { t } = useTranslation()
  const theme = useUiStore((s) => s.theme)
  const cycleTheme = useUiStore((s) => s.cycleTheme)
  const lang = useUiStore((s) => s.lang)
  const setLang = useUiStore((s) => s.setLang)
  const logout = useLogout()
  const ThemeIcon = THEME_ICON[theme]

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-1 border-b bg-background px-3 md:h-16 md:px-4">
      <div className="flex min-w-0 items-center gap-1">
        {onOpenNav && (
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label={t('nav.menu')}
            onClick={onOpenNav}
          >
            <Menu className="size-5" />
          </Button>
        )}
        <span className="truncate text-sm font-semibold tracking-tight md:hidden">
          {t('app.title')}
        </span>
      </div>

      <div className="flex items-center gap-0.5 sm:gap-1">
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
      </div>
    </header>
  )
}
