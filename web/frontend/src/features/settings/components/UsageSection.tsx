// UsageSection — the allow-over-usage toggle. Loads core settings, saves via
// useUpdateSettings.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useSettings } from '@/hooks/queries/useSettings'
import { useUpdateSettings } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'

export function UsageSection() {
  const { t } = useTranslation()
  const settings = useSettings()
  const save = useUpdateSettings()
  const [allow, setAllow] = useState(false)
  const [multiplier, setMultiplier] = useState('0')

  useEffect(() => {
    if (settings.data) {
      setAllow(settings.data.allowOverUsage)
      setMultiplier(String(settings.data.defaultApiKeyMultiplier ?? 0))
    }
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
          <div className="space-y-2">
            <Label>Default multiplier for new API keys</Label>
            <Input type="number" min={0} max={100} step={0.1} value={multiplier} onChange={(e) => setMultiplier(e.target.value)} />
            <p className="text-sm text-muted-foreground">0 = default 1x; new keys use this value unless explicitly configured.</p>
          </div>
          <Button
            disabled={save.isPending}
            onClick={() =>
              save.mutate(
                { allowOverUsage: allow, defaultApiKeyMultiplier: Number(multiplier) || 0 },
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
