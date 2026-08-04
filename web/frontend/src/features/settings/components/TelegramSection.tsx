// TelegramSection — enable/disable + bot token (masked, write-only) + chat id.
// Token field is left blank on load (backend returns only a masked hint); sending
// an empty token leaves it unchanged.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/shared/PasswordInput'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useTelegram } from '@/hooks/queries/useSettings'
import { useUpdateTelegram, useTestTelegram } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'
import { SettingsToggleRow } from './SettingsToggleRow'

export function TelegramSection() {
  const { t } = useTranslation()
  const telegram = useTelegram()
  const save = useUpdateTelegram()
  const test = useTestTelegram()

  const [enabled, setEnabled] = useState(false)
  const [chatId, setChatId] = useState('')
  const [token, setToken] = useState('')

  useEffect(() => {
    if (telegram.data) {
      setEnabled(telegram.data.enabled)
      setChatId(telegram.data.chatId)
    }
  }, [telegram.data])

  function onSave() {
    save.mutate(
      { enabled, chatId, botToken: token || undefined },
      {
        onSuccess: () => {
          toast.success(t('settings.telegramSaved'))
          setToken('')
        },
        onError: () => toast.error(t('common.saveFailed')),
      },
    )
  }

  return (
    <SettingsSection id="telegram" title={t('settings.telegramSettings')} description={t('settings.telegramDesc')}>
      {telegram.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <SettingsToggleRow
            label={t('settings.telegramEnabled')}
            hint={t('settings.telegramEnabledHint')}
            checked={enabled}
            onChange={setEnabled}
          />
          <div className="space-y-2">
            <Label>{t('settings.telegramBotToken')}</Label>
            <PasswordInput
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={telegram.data?.botTokenSet ? telegram.data.botTokenMasked : ''}
              autoComplete="off"
            />
            <p className="text-sm text-muted-foreground">{t('settings.telegramBotTokenHint')}</p>
          </div>
          <div className="space-y-2">
            <Label>{t('settings.telegramChatId')}</Label>
            <Input value={chatId} onChange={(e) => setChatId(e.target.value)} />
            <p className="text-sm text-muted-foreground">{t('settings.telegramChatIdHint')}</p>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button className="w-full sm:w-auto" disabled={save.isPending} onClick={onSave}>
              {t('settings.saveTelegram')}
            </Button>
            <Button
              className="w-full sm:w-auto"
              variant="outline"
              disabled={test.isPending}
              onClick={() =>
                test.mutate(undefined, {
                  onSuccess: () => toast.success(t('settings.telegramTestOk')),
                  onError: () => toast.error(t('settings.telegramTestFailed')),
                })
              }
            >
              {t('settings.testTelegram')}
            </Button>
          </div>
        </>
      )}
    </SettingsSection>
  )
}
