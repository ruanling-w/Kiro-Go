// UsageSection — the allow-over-usage toggle. Loads core settings, saves via
// useUpdateSettings.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useSettings } from '@/hooks/queries/useSettings'
import { useUpdateSettings } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'

export function UsageSection() {
  const { t } = useTranslation()
  const settings = useSettings()
  const save = useUpdateSettings()
  const [allow, setAllow] = useState(false)

  useEffect(() => {
    if (settings.data) setAllow(settings.data.allowOverUsage)
  }, [settings.data])

  return (
    <SettingsSection id="usage" title={t('settings.usageSettings')} description={t('settings.usageDesc')}>
      {settings.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="flex items-center justify-between gap-4">
            <div>
              <Label>{t('settings.allowOverUsage')}</Label>
              <p className="mt-1 text-sm text-muted-foreground">{t('settings.allowOverUsageHint')}</p>
            </div>
            <Switch checked={allow} onCheckedChange={setAllow} />
          </div>
          <Button
            disabled={save.isPending}
            onClick={() =>
              save.mutate(
                { allowOverUsage: allow },
                {
                  onSuccess: () => toast.success(t('settings.overUsageSaved')),
                  onError: () => toast.error(t('common.saveFailed')),
                },
              )
            }
          >
            {t('settings.saveUsage')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}
