// LoginPage — password → cookie session. No password is retained after login.
//
// On submit we POST /login; the backend sets the HttpOnly session + readable CSRF
// cookies and we flip authStore.authed. A 429 means the IP is locked out (too
// many failed attempts) — we surface the Retry-After hint. A non-HTTPS, non-local
// origin shows a warning banner since the password crosses the wire in cleartext.
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Languages, Moon, Sun, MonitorCog, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import { useLogin } from '@/hooks/useAuth'
import { useUiStore, type ThemePref } from '@/stores/uiStore'
import { LANGS, type Lang } from '@/lib/i18n'
import { ApiError } from '@/services/httpClient'

const THEME_ICON: Record<ThemePref, typeof Sun> = {
  system: MonitorCog,
  light: Sun,
  dark: Moon,
}

function isInsecureContext(): boolean {
  if (window.location.protocol === 'https:') return false
  const h = window.location.hostname
  return h !== 'localhost' && h !== '127.0.0.1' && h !== '::1'
}

export function LoginPage() {
  const { t } = useTranslation()
  const login = useLogin()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const theme = useUiStore((s) => s.theme)
  const cycleTheme = useUiStore((s) => s.cycleTheme)
  const lang = useUiStore((s) => s.lang)
  const setLang = useUiStore((s) => s.setLang)
  const ThemeIcon = THEME_ICON[theme]

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!password) {
      setError(t('login.passwordRequired'))
      return
    }
    login.mutate(password, {
      onError: (err) => {
        if (err instanceof ApiError && err.status === 429) {
          setError(t('login.error'))
        } else if (err instanceof ApiError && err.status === 401) {
          setError(t('login.error'))
        } else {
          setError(t('login.connectError'))
        }
        setPassword('')
      },
    })
  }

  return (
    <div className="relative flex min-h-dvh items-center justify-center overflow-hidden bg-background px-4">
      {/* Ambient teal field — static radial washes, no motion, so the login
          reads as an instrument surface without distracting from the form. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 opacity-70 dark:opacity-50"
        style={{
          background:
            'radial-gradient(60% 50% at 15% 20%, var(--glow), transparent 70%), radial-gradient(50% 45% at 85% 85%, oklch(0.66 0.13 220 / 30%), transparent 70%)',
        }}
      />
      <div className="absolute right-4 top-4 flex items-center gap-1">
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
      </div>

      <div className="w-full max-w-sm rounded-2xl border bg-card/90 p-8 shadow-[0_20px_60px_-24px_var(--glow)] backdrop-blur-sm ring-1 ring-primary/10">
        <div className="mb-6 flex flex-col items-center text-center">
          <div className="mb-3 flex size-24 items-center justify-center rounded-3xl bg-[image:var(--brand-gradient)] p-1 shadow-[0_8px_28px_-8px_var(--glow)]">
            <div className="flex size-full items-center justify-center rounded-[calc(var(--radius)*2)] bg-card">
              <img
                src="/admin/logo.png"
                alt=""
                className="size-16 rounded-2xl object-contain"
                onError={(e) => (e.currentTarget.style.display = 'none')}
              />
            </div>
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">{t('app.title')}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t('login.subtitle')}</p>
        </div>

        {isInsecureContext() && (
          <div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400">
            <ShieldAlert className="mt-0.5 size-4 shrink-0" />
            <span>{t('login.secureNote')}</span>
          </div>
        )}

        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="password">{t('login.password')}</Label>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('login.passwordPlaceholder')}
              autoComplete="current-password"
              autoFocus
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <Button type="submit" className="w-full" disabled={login.isPending}>
            {login.isPending ? (
              <span className="flex items-center gap-2">
                <HamsterWheel size="sm" />
                {t('login.submitting')}
              </span>
            ) : (
              t('login.submit')
            )}
          </Button>
        </form>
      </div>
    </div>
  )
}
