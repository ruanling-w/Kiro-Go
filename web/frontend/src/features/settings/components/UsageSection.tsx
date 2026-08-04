// UsageSection — server-wide usage controls: allow-over-usage, global API-key
// billing multiplier, and global rolling-60s RPM cap (sum of all keys).
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useSettings } from '@/hooks/queries/useSettings'
import { useUpdateSettings } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'
import { SettingsToggleRow } from './SettingsToggleRow'

export function UsageSection() {
  const { t } = useTranslation()
  const settings = useSettings()
  const save = useUpdateSettings()
  const [allow, setAllow] = useState(false)
  const [multiplier, setMultiplier] = useState('0')
  const [rpmLimit, setRpmLimit] = useState('0')

  useEffect(() => {
    if (settings.data) {
      setAllow(settings.data.allowOverUsage)
      setMultiplier(String(settings.data.defaultApiKeyMultiplier ?? 0))
      setRpmLimit(String(settings.data.defaultApiKeyRpmLimit ?? 0))
    }
  }, [settings.data])

  return (
    <SettingsSection
      id="usage"
      title={t('settings.usageSettings')}
      description={t('settings.usageDesc')}
    >
      {settings.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <SettingsToggleRow
            label={t('settings.allowOverUsage')}
            hint={t('settings.allowOverUsageHint')}
            checked={allow}
            onChange={setAllow}
          />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="min-w-0 space-y-2">
              <Label htmlFor="usage-multiplier" className="block whitespace-normal leading-snug">
                {t('settings.serverMultiplier')}
              </Label>
              <Input
                id="usage-multiplier"
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={multiplier}
                onChange={(e) => setMultiplier(e.target.value)}
                className="min-w-0"
              />
              <p className="text-sm leading-relaxed text-muted-foreground text-pretty">
                {t('settings.serverMultiplierHint')}
              </p>
            </div>

            <div className="min-w-0 space-y-2">
              <Label htmlFor="usage-rpm" className="block whitespace-normal leading-snug">
                {t('settings.serverRpmLimit')}
              </Label>
              <Input
                id="usage-rpm"
                type="number"
                min={0}
                step={1}
                value={rpmLimit}
                onChange={(e) => setRpmLimit(e.target.value)}
                className="min-w-0"
              />
              <p className="text-sm leading-relaxed text-muted-foreground text-pretty">
                {t('settings.serverRpmLimitHint')}
              </p>
            </div>
          </div>

          <Button
            className="w-full sm:w-auto"
            disabled={save.isPending}
            onClick={() =>
              save.mutate(
                {
                  allowOverUsage: allow,
                  defaultApiKeyMultiplier: Number(multiplier) || 0,
                  defaultApiKeyRpmLimit: Math.max(0, Math.floor(Number(rpmLimit) || 0)),
                },
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
