// CheckPage — public standalone key-lookup page (no admin auth). A key holder
// pastes their key and sees quota/lifetime/usage. Reuses the app's Tailwind
// tokens + theme; strings are Vietnamese-first per the legacy page.
import { useState, type FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { KeyRound, Moon, Sun } from 'lucide-react'
import { lookupKey, type CheckKeyResponse } from '@/services/checkKey.service'
import { ApiError } from '@/services/httpClient'
import { Button } from '@/components/ui/button'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { UsageBar } from '@/components/shared/UsageBar'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import { formatNumber, formatUnixSeconds } from '@/lib/format'

function toggleTheme() {
  const root = document.documentElement
  const dark = root.classList.toggle('dark')
  localStorage.setItem('kiro_theme', dark ? 'dark' : 'light')
}

export function CheckPage() {
  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  const lookup = useMutation({
    mutationFn: (k: string) => lookupKey(k),
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Không thể kiểm tra API key')
    },
  })

  const data = lookup.data as CheckKeyResponse | undefined

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    if (!key.trim()) {
      setError('Vui lòng nhập API key')
      return
    }
    lookup.mutate(key.trim())
  }

  return (
    <div className="min-h-dvh bg-background px-4 py-10 text-foreground">
      <div className="mx-auto w-full max-w-2xl">
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <KeyRound className="size-6 text-primary" />
            <h1 className="text-xl font-semibold">Kiểm tra API Key</h1>
          </div>
          <Button variant="ghost" size="icon" aria-label="Theme" onClick={toggleTheme}>
            <Sun className="size-5 dark:hidden" />
            <Moon className="hidden size-5 dark:block" />
          </Button>
        </div>

        <form onSubmit={onSubmit} className="flex gap-2">
          <PasswordInput
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="Nhập API key của bạn"
            autoFocus
          />
          <Button type="submit" disabled={lookup.isPending}>
            {lookup.isPending ? <HamsterWheel size="sm" /> : 'Kiểm tra'}
          </Button>
        </form>

        {error && <p className="mt-3 text-sm text-destructive">{error}</p>}

        {data && (
          <div className="mt-6 space-y-4">
            <div className="rounded-xl border bg-card p-5">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">{data.name || 'API Key'}</p>
                  <p className="font-mono text-sm text-muted-foreground">{data.keyMasked}</p>
                </div>
                <span
                  className={
                    data.expired || !data.enabled
                      ? 'rounded-full bg-destructive/10 px-3 py-1 text-sm text-destructive'
                      : 'rounded-full bg-emerald-500/10 px-3 py-1 text-sm text-emerald-600 dark:text-emerald-400'
                  }
                >
                  {data.expired ? 'Hết hạn' : data.enabled ? 'Hoạt động' : 'Vô hiệu'}
                </span>
              </div>

              <div className="mt-4 space-y-3">
                {!data.creditUnlimited && (
                  <UsageBar
                    label="Credits"
                    used={data.creditsUsed}
                    limit={data.creditLimit}
                  />
                )}
                {!data.tokenUnlimited && (
                  <UsageBar label="Tokens" used={data.tokensUsed} limit={data.tokenLimit} />
                )}
              </div>

              <dl className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
                <Field label="Requests" value={formatNumber(data.requestsCount)} />
                <Field
                  label="Credits"
                  value={data.creditUnlimited ? '∞' : formatNumber(data.creditsRemaining)}
                />
                <Field
                  label="Tokens"
                  value={data.tokenUnlimited ? '∞' : formatNumber(data.tokensRemaining)}
                />
                <Field
                  label="Hết hạn"
                  value={data.neverExpires ? 'Vĩnh viễn' : formatUnixSeconds(data.expiresAt, { dateStyle: 'short' })}
                />
                <Field label="Còn lại" value={data.neverExpires ? '∞' : `${data.daysRemaining} ngày`} />
                <Field label="Tạo lúc" value={formatUnixSeconds(data.createdAt, { dateStyle: 'short' })} />
              </dl>
            </div>

            {data.logs.length > 0 && (
              <div className="rounded-xl border bg-card p-5">
                <p className="mb-3 font-medium">Lịch sử ({data.logs.length})</p>
                <div className="max-h-96 space-y-1 overflow-auto font-mono text-xs">
                  {data.logs.map((log, i) => (
                    <div
                      key={i}
                      className="flex items-center justify-between gap-2 border-b py-1.5 last:border-0"
                    >
                      <span className="text-muted-foreground">
                        {formatUnixSeconds(log.time, { dateStyle: 'short', timeStyle: 'short' })}
                      </span>
                      <span className="truncate">{log.model || log.endpoint}</span>
                      <span
                        className={
                          log.status === 'success' ? 'text-emerald-500' : 'text-destructive'
                        }
                      >
                        {formatNumber(log.tokens)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  )
}
