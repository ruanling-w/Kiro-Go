// CheckPage — public standalone key-lookup mini-dashboard (no admin auth).
// Flow: enter key → sessionStorage + useQuery → KPI / quota / chart / logs.
// Auto-refreshes every 30s while a key is held in the tab.
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Activity,
  CalendarClock,
  Coins,
  KeyRound,
  Moon,
  ShieldCheck,
  Sun,
} from 'lucide-react'
import { lookupKey } from '@/services/checkKey.service'
import { ApiError } from '@/services/httpClient'
import { Button } from '@/components/ui/button'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useCheckKeySession } from './hooks/useCheckKeySession'
import { CheckKeyForm } from './components/CheckKeyForm'
import { CheckDashboard } from './components/CheckDashboard'

const REFETCH_MS = 30_000

function toggleTheme() {
  const root = document.documentElement
  const dark = root.classList.toggle('dark')
  localStorage.setItem('kiro_theme', dark ? 'dark' : 'light')
}

export function CheckPage() {
  const { t } = useTranslation()
  const { key, setKey, clear } = useCheckKeySession()
  const [formError, setFormError] = useState('')

  const query = useQuery({
    queryKey: ['check-key', key],
    queryFn: () => lookupKey(key!),
    enabled: !!key,
    refetchInterval: key ? REFETCH_MS : false,
    retry: false,
    staleTime: 10_000,
  })

  // Drop a stored key that the server says does not exist so the form returns.
  // Keep the not-found message on the form after clear().
  useEffect(() => {
    if (!key || !query.isError) return
    if (query.error instanceof ApiError && query.error.status === 404) {
      setFormError(t('check.error.notFound'))
      clear()
    }
  }, [key, query.isError, query.error, clear, t])

  const errorMessage = useMemo(() => {
    if (formError) return formError
    if (!query.isError) return ''
    const err = query.error
    if (err instanceof ApiError) {
      if (err.status === 0) return t('check.error.network')
      if (err.status === 404) return t('check.error.notFound')
      return err.message || t('check.error.generic')
    }
    return t('check.error.generic')
  }, [formError, query.isError, query.error, t])

  function onSubmit(raw: string) {
    setFormError('')
    if (!raw) {
      setFormError(t('check.error.empty'))
      return
    }
    setKey(raw)
  }

  function onClear() {
    setFormError('')
    clear()
  }

  const showDashboard = !!key && !!query.data
  const showLoading = !!key && query.isPending && !query.data
  // Keep the form visible until we have dashboard data (and after clear / 404).
  const showForm = !showDashboard && !showLoading

  return (
    <div className="relative min-h-dvh overflow-hidden bg-background text-foreground">
      {/* Ambient teal field — matches admin login. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 opacity-70 dark:opacity-50"
        style={{
          background:
            'radial-gradient(55% 45% at 18% 12%, var(--glow), transparent 70%), radial-gradient(50% 45% at 88% 92%, oklch(0.66 0.13 220 / 28%), transparent 70%)',
        }}
      />

      {/* Top bar — brand + theme (dashboard keeps a compact header inside). */}
      <header className="mx-auto flex w-full max-w-6xl items-center justify-between gap-3 px-4 py-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-2.5">
          <img
            src="/admin/logo.png"
            alt=""
            className="size-8 shrink-0 object-contain sm:size-9"
            onError={(e) => {
              e.currentTarget.style.display = 'none'
            }}
          />
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold tracking-tight">
              {t('app.title')}
            </p>
            {showDashboard && (
              <p className="truncate text-xs text-muted-foreground">
                {t('check.title')}
              </p>
            )}
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="shrink-0"
          aria-label={t('check.theme')}
          onClick={toggleTheme}
        >
          <Sun className="size-5 dark:hidden" />
          <Moon className="hidden size-5 dark:block" />
        </Button>
      </header>

      <main
        className={
          showForm || showLoading
            ? 'flex min-h-[calc(100dvh-4.5rem)] items-center justify-center px-4 pb-10 pt-2'
            : 'mx-auto w-full max-w-6xl min-w-0 px-3 pb-10 pt-1 sm:px-6'
        }
      >
        {showForm && (
          <div className="w-full max-w-md">
            <div className="rounded-2xl border bg-card/85 p-6 shadow-[0_24px_70px_-28px_var(--glow)] ring-1 ring-primary/10 backdrop-blur-md sm:p-8">
              <div className="mb-6 flex flex-col items-center text-center">
                <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-primary/10 text-primary ring-1 ring-primary/15">
                  <KeyRound className="size-7" />
                </div>
                <h1 className="text-2xl font-semibold tracking-tight">
                  {t('check.title')}
                </h1>
                <p className="mt-2 max-w-sm text-sm leading-relaxed text-muted-foreground text-pretty">
                  {t('check.subtitle')}
                </p>
              </div>

              <CheckKeyForm
                onSubmit={onSubmit}
                pending={!!key && query.isFetching}
                error={errorMessage}
              />

              <ul className="mt-6 grid grid-cols-1 gap-2 sm:grid-cols-3">
                <Highlight
                  icon={CalendarClock}
                  label={t('check.highlight.expiry')}
                />
                <Highlight icon={Coins} label={t('check.highlight.quota')} />
                <Highlight icon={Activity} label={t('check.highlight.history')} />
              </ul>

              <p className="mt-5 flex items-start justify-center gap-1.5 text-center text-[11px] leading-relaxed text-muted-foreground">
                <ShieldCheck className="mt-0.5 size-3.5 shrink-0 text-primary/80" />
                <span>{t('check.privacyNote')}</span>
              </p>
            </div>
          </div>
        )}

        {showLoading && (
          <div className="w-full max-w-md rounded-2xl border bg-card/85 p-10 shadow-[0_24px_70px_-28px_var(--glow)] ring-1 ring-primary/10 backdrop-blur-md">
            <HamsterLoader label={t('check.checking')} />
          </div>
        )}

        {showDashboard && query.data && (
          <CheckDashboard
            data={query.data}
            updatedAt={
              query.dataUpdatedAt ? Math.floor(query.dataUpdatedAt / 1000) : undefined
            }
            refreshing={query.isFetching && !query.isPending}
            onRefresh={() => {
              void query.refetch()
            }}
            onClear={onClear}
          />
        )}

        {!showForm && query.isError && errorMessage && (
          <p className="mt-4 text-sm text-destructive" role="alert">
            {errorMessage}
          </p>
        )}
      </main>
    </div>
  )
}

function Highlight({
  icon: Icon,
  label,
}: {
  icon: typeof CalendarClock
  label: string
}) {
  return (
    <li className="flex items-center gap-2 rounded-xl border border-border/60 bg-muted/30 px-3 py-2 text-left">
      <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Icon className="size-3.5" />
      </span>
      <span className="text-xs font-medium leading-snug text-foreground/90">
        {label}
      </span>
    </li>
  )
}
