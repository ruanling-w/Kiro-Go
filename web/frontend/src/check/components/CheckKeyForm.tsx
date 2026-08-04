// CheckKeyForm — public key entry. Password-masked input + submit, styled for
// the centered landing card.
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { HamsterWheel } from '@/components/shared/HamsterLoader'
import { cn } from '@/lib/utils'

interface CheckKeyFormProps {
  onSubmit: (key: string) => void
  pending?: boolean
  error?: string
  initialValue?: string
}

export function CheckKeyForm({
  onSubmit,
  pending,
  error,
  initialValue = '',
}: CheckKeyFormProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialValue)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    onSubmit(value.trim())
  }

  return (
    <form onSubmit={handleSubmit} className="min-w-0 space-y-4">
      <div className="min-w-0 space-y-2">
        <Label htmlFor="check-api-key" className="text-sm font-medium">
          {t('check.keyLabel')}
        </Label>
        <PasswordInput
          id="check-api-key"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('check.placeholder')}
          autoFocus
          autoComplete="off"
          spellCheck={false}
          disabled={pending}
          aria-invalid={!!error || undefined}
          className={cn(
            'h-11 min-w-0 font-mono text-sm tracking-wide md:text-sm',
            error && 'border-destructive',
          )}
        />
        {error ? (
          <p className="text-sm text-destructive" role="alert">
            {error}
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">{t('check.keyHint')}</p>
        )}
      </div>

      <Button
        type="submit"
        disabled={pending}
        size="lg"
        className="h-11 w-full gap-2 text-sm font-semibold"
      >
        {pending ? (
          <HamsterWheel size="sm" />
        ) : (
          <>
            <Search className="size-4" />
            {t('check.submit')}
          </>
        )}
      </Button>
    </form>
  )
}
