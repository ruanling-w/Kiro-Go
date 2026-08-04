// CheckKeyForm — public key entry. Password-masked input + submit.
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { HamsterWheel } from '@/components/shared/HamsterLoader'

interface CheckKeyFormProps {
  onSubmit: (key: string) => void
  pending?: boolean
  error?: string
  initialValue?: string
}

export function CheckKeyForm({ onSubmit, pending, error, initialValue = '' }: CheckKeyFormProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(initialValue)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    onSubmit(value.trim())
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div className="flex flex-col gap-2 sm:flex-row">
        <div className="min-w-0 sm:flex-1">
          <PasswordInput
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder={t('check.placeholder')}
            autoFocus
            autoComplete="off"
            spellCheck={false}
            className="font-mono"
            disabled={pending}
          />
        </div>
        <Button type="submit" disabled={pending} className="sm:min-w-28">
          {pending ? <HamsterWheel size="sm" /> : t('check.submit')}
        </Button>
      </div>
      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
    </form>
  )
}
