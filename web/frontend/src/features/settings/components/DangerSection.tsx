// DangerSection — reset the aggregate statistics counters. Guarded by the shared
// ConfirmDialog (never native confirm()).
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { useResetStats } from '@/hooks/mutations/useSettingsMutations'
import { SettingsSection } from './SettingsSection'
import { SettingsToggleRow } from './SettingsToggleRow'

export function DangerSection() {
  const { t } = useTranslation()
  const reset = useResetStats()
  const [open, setOpen] = useState(false)

  return (
    <SettingsSection
      id="danger"
      title={t('settings.dangerTitle')}
      description={t('settings.dangerHint')}
    >
      <SettingsToggleRow
        label={t('settings.statistics')}
        hint={t('settings.statsDesc')}
        control={
          <Button
            variant="destructive"
            className="whitespace-normal sm:whitespace-nowrap"
            onClick={() => setOpen(true)}
            disabled={reset.isPending}
          >
            {t('settings.resetStats')}
          </Button>
        }
      />

      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={t('settings.confirmReset')}
        confirmLabel={t('settings.resetStats')}
        destructive
        onConfirm={() =>
          reset.mutate(undefined, {
            onSuccess: () => toast.success(t('settings.statsReset')),
            onError: () => toast.error(t('common.failed')),
          })
        }
      />
    </SettingsSection>
  )
}
