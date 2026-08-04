// ApiDocsPage — single admin page: connection picker + endpoint table +
// CLI/SDK guides + live model catalog. Snippets fill from connection state.
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '@/components/shared/PageHeader'
import { ConnectionCard, type ConnectionValues } from './components/ConnectionCard'
import { EndpointTable } from './components/EndpointTable'
import { ToolGuideTabs } from './components/ToolGuideTabs'
import { ModelCatalogCard } from './components/ModelCatalogCard'
import type { PublicModel } from '@/types/publicApi'

const EMPTY: ConnectionValues = {
  base: '',
  key: '',
  model: '',
  models: [] as PublicModel[],
  modelsLoading: true,
  modelsError: false,
}

export default function ApiDocsPage() {
  const { t } = useTranslation()
  const [conn, setConn] = useState<ConnectionValues>(EMPTY)
  const handleConn = useCallback((v: ConnectionValues) => setConn(v), [])

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('apiDocs.title')}
        description={t('apiDocs.subtitle')}
      />

      <ConnectionCard onChange={handleConn} />

      <div className="grid gap-6 xl:grid-cols-5">
        <div className="space-y-6 xl:col-span-3">
          <EndpointTable base={conn.base} />
          <ToolGuideTabs
            vars={{
              base: conn.base,
              key: conn.key,
              model: conn.model,
            }}
          />
        </div>
        <div className="xl:col-span-2">
          <ModelCatalogCard
            models={conn.models}
            loading={conn.modelsLoading}
            error={conn.modelsError}
          />
        </div>
      </div>
    </div>
  )
}
