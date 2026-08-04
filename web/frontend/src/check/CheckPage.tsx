// CheckPage — public standalone key-lookup mini-dashboard (no admin auth).
// Flow: enter key → sessionStorage + useQuery → KPI / quota / chart / logs.
// Auto-refreshes every 30s while a key is held in the tab.
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { KeyRound, Moon, Sun } from 'lucide-react'
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
    <div className="min-h-dvh bg-background px-4 py-10 text-foreground">
      <div className={`mx-auto w-full ${showDashboard ? 'max-w-6xl' : 'max-w-2xl'}`}>
        <div className="mb-6 flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <KeyRound className="size-6 shrink-0 text-primary" />
            <div className="min-w-0">
              <h1 className="text-xl font-semibold">{t('check.title')}</h1>
              {!showDashboard && (
                <p className="mt-0.5 text-sm text-muted-foreground">{t('check.subtitle')}</p>
              )}
            </div>
          </div>
          <Button variant="ghost" size="icon" aria-label={t('check.theme')} onClick={toggleTheme}>
            <Sun className="size-5 dark:hidden" />
            <Moon className="hidden size-5 dark:block" />
          </Button>
        </div>

        {showForm && (
          <CheckKeyForm
            onSubmit={onSubmit}
            pending={!!key && query.isFetching}
            error={errorMessage}
          />
        )}

        {showLoading && (
          <div className="mt-10">
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
      </div>
    </div>
  )
}
