// ApiEndpointsSection — short pointer to the dedicated API docs page, plus the
// existing stats dialog (kept so Settings scroll-spy id="api" still works).
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { HamsterLoader } from '@/components/shared/HamsterLoader'
import { useStats } from '@/hooks/queries/useStatus'
import { SettingsSection } from './SettingsSection'

export function ApiEndpointsSection() {
  const { t } = useTranslation()
  const [viewStats, setViewStats] = useState(false)
  const stats = useStats()

  return (
    <SettingsSection id="api" title={t('nav.usage')} description={t('tabs.api')}>
      <p className="text-sm text-muted-foreground">{t('apiDocs.settingsPointer')}</p>

      <div className="flex flex-wrap gap-2">
        <Button asChild variant="default" size="sm">
          <Link to="/api-docs">
            <BookOpen className="size-4" />
            {t('apiDocs.openPage')}
          </Link>
        </Button>
        <Button variant="outline" size="sm" onClick={() => setViewStats(true)}>
          {t('settings.statistics')}
        </Button>
      </div>

      <Dialog open={viewStats} onOpenChange={setViewStats}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('settings.statistics')}</DialogTitle>
          </DialogHeader>
          {stats.isPending ? (
            <HamsterLoader size="sm" />
          ) : (
            <pre className="max-h-[60vh] overflow-auto rounded-lg bg-muted p-4 text-xs">
              {JSON.stringify(stats.data, null, 2)}
            </pre>
          )}
        </DialogContent>
      </Dialog>
    </SettingsSection>
  )
}
