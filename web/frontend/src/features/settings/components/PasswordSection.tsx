// PasswordSection — change the admin password. On success the backend revokes
// all sessions, so we log out locally and bounce to login.
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { useUpdateSettings } from '@/hooks/mutations/useSettingsMutations'
import { forceLogout } from '@/stores/authStore'
import { SettingsSection } from './SettingsSection'

export function PasswordSection() {
  const { t } = useTranslation()
  const save = useUpdateSettings()
  const [password, setPassword] = useState('')

  function onSave() {
    if (!password) {
      toast.error(t('settings.passwordRequired'))
      return
    }
    save.mutate(
      { password },
      {
        onSuccess: () => {
          toast.success(t('settings.passwordChanged'))
          setPassword('')
          // Backend revoked all sessions — force re-login.
          setTimeout(() => forceLogout(), 800)
        },
        onError: () => toast.error(t('common.saveFailed')),
      },
    )
  }

  return (
    <SettingsSection id="password" title={t('settings.adminPassword')} description={t('settings.passwordDesc')}>
      <div className="space-y-2">
        <Label htmlFor="new-password">{t('settings.newPassword')}</Label>
        <PasswordInput
          id="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t('settings.newPasswordPlaceholder')}
          autoComplete="new-password"
        />
      </div>
      <Button disabled={save.isPending} onClick={onSave}>
        {t('settings.changePassword')}
      </Button>
    </SettingsSection>
  )
}
