// ApiAccessCard — toggles requireApiKey (global gate) and shows the warning when
// enforcement is on but no enabled key exists (would lock everyone out).
import { useTranslation } from 'react-i18next'
import { ShieldCheck, TriangleAlert } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { useSettings } from '@/hooks/queries/useSettings'
import { useUpdateSettings } from '@/hooks/mutations/useSettingsMutations'
import { toast } from 'sonner'

export function ApiAccessCard({ hasEnabledKey }: { hasEnabledKey: boolean }) {
  const { t } = useTranslation()
  const settings = useSettings()
  const update = useUpdateSettings()
  const require = settings.data?.requireApiKey ?? false

  function toggle(v: boolean) {
    update.mutate(
      { requireApiKey: v },
      {
        onSuccess: () => toast.success(t('common.saved')),
        onError: () => toast.error(t('common.saveFailed')),
      },
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ShieldCheck className="size-5" />
          {t('apiKeys.listTitle')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between gap-4">
          <p className="text-sm text-muted-foreground">{t('apiKeys.requireHint')}</p>
          <Switch checked={require} onCheckedChange={toggle} disabled={update.isPending} />
        </div>
        {require && !hasEnabledKey && (
          <div className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400">
            <TriangleAlert className="mt-0.5 size-4 shrink-0" />
            <span>{t('apiKeys.requireWithoutEnabledKeyWarning')}</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
