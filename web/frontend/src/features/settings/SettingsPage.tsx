// Settings — scroll-spy sidebar + stacked config sections. Each section is a
// self-contained card that loads its own config slice and saves via its own
// mutation. The "API endpoints" section (merged from the old API view) lists the
// copyable client endpoints + a models/stats viewer.
import { useTranslation } from 'react-i18next'
import { PageHeader } from '@/components/shared/PageHeader'
import { SettingsNav } from './components/SettingsNav'
import { UsageSection } from './components/UsageSection'
import { ThinkingSection } from './components/ThinkingSection'
import { EndpointSection } from './components/EndpointSection'
import { TelegramSection } from './components/TelegramSection'
import { ProxySection } from './components/ProxySection'
import { PromptFilterSection } from './components/PromptFilterSection'
import { PasswordSection } from './components/PasswordSection'
import { DangerSection } from './components/DangerSection'
import { ApiEndpointsSection } from './components/ApiEndpointsSection'

export default function SettingsPage() {
  const { t } = useTranslation()
  return (
    <div className="space-y-6">
      <PageHeader title={t('nav.system')} description={t('settings.pageSubtitle')} />
      <div className="flex flex-col gap-6 lg:flex-row">
        <SettingsNav />
        <div className="min-w-0 flex-1 space-y-6">
          <UsageSection />
          <ThinkingSection />
          <EndpointSection />
          <TelegramSection />
          <ProxySection />
          <PromptFilterSection />
          <ApiEndpointsSection />
          <PasswordSection />
          <DangerSection />
        </div>
      </div>
    </div>
  )
}
