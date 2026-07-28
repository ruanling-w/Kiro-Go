// EndpointSection — preferred client endpoint + fallback toggle.
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useEndpoint } from '@/hooks/queries/useSettings'
import { useUpdateEndpoint } from '@/hooks/mutations/useSettingsMutations'
import type { PreferredEndpoint } from '@/types/settings'
import { SettingsSection } from './SettingsSection'

const ENDPOINTS: { value: PreferredEndpoint; key: string }[] = [
  { value: 'auto', key: 'settings.endpointAuto' },
  { value: 'kiro', key: 'settings.endpointKiro' },
  { value: 'codewhisperer', key: 'settings.endpointCodeWhisperer' },
  { value: 'amazonq', key: 'settings.endpointAmazonQ' },
]

export function EndpointSection() {
  const { t } = useTranslation()
  const endpoint = useEndpoint()
  const save = useUpdateEndpoint()
  const [preferred, setPreferred] = useState<PreferredEndpoint>('auto')
  const [fallback, setFallback] = useState(true)

  useEffect(() => {
    if (endpoint.data) {
      setPreferred(endpoint.data.preferredEndpoint)
      setFallback(endpoint.data.endpointFallback)
    }
  }, [endpoint.data])

  return (
    <SettingsSection id="endpoint" title={t('settings.endpointSettings')} description={t('settings.endpointDesc')}>
      {endpoint.isPending ? (
        <HamsterLoader size="sm" />
      ) : (
        <>
          <div className="space-y-2">
            <Label>{t('settings.preferredEndpoint')}</Label>
            <Select value={preferred} onValueChange={(v) => setPreferred(v as PreferredEndpoint)}>
              <SelectTrigger className="w-full sm:w-64">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ENDPOINTS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {t(o.key)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-sm text-muted-foreground">{t('settings.endpointHint')}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <div>
              <Label>{t('settings.endpointFallback')}</Label>
              <p className="mt-1 text-sm text-muted-foreground">{t('settings.endpointFallbackHint')}</p>
            </div>
            <Switch checked={fallback} onCheckedChange={setFallback} />
          </div>
          <Button
            disabled={save.isPending}
            onClick={() =>
              save.mutate(
                { preferredEndpoint: preferred, endpointFallback: fallback },
                {
                  onSuccess: () => toast.success(t('settings.endpointSaved')),
                  onError: () => toast.error(t('common.saveFailed')),
                },
              )
            }
          >
            {t('settings.saveEndpoint')}
          </Button>
        </>
      )}
    </SettingsSection>
  )
}
