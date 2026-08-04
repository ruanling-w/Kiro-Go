// EndpointTable — static catalog of client-facing gateway routes.
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { CopyButton } from '@/components/shared/CopyButton'
import { DataTable, type Column } from '@/components/shared/DataTable'
import { API_ENDPOINTS, type ApiEndpointDoc } from '@/config/apiDocs'

interface EndpointTableProps {
  base: string
}

export function EndpointTable({ base }: EndpointTableProps) {
  const { t } = useTranslation()
  const origin = (base || '').replace(/\/+$/, '')

  const columns = useMemo<Column<ApiEndpointDoc>[]>(
    () => [
      {
        id: 'method',
        header: t('apiDocs.col.method'),
        // Folded into the mobile primary path row.
        mobileHidden: true,
        cell: (row) => (
          <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs font-semibold text-muted-foreground">
            {row.method}
          </span>
        ),
        sortValue: (row) => row.method,
        className: 'w-20',
      },
      {
        id: 'path',
        header: t('apiDocs.col.path'),
        mobileRole: 'primary',
        cell: (row) => (
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs font-semibold text-muted-foreground md:hidden">
                {row.method}
              </span>
              <p className="break-all font-mono text-sm">{row.path}</p>
            </div>
            {row.aliases?.length ? (
              <p className="truncate text-xs text-muted-foreground">
                {row.aliases.join(' · ')}
              </p>
            ) : null}
          </div>
        ),
        sortValue: (row) => row.path,
      },
      {
        id: 'auth',
        header: t('apiDocs.col.auth'),
        mobileRole: 'badge',
        cell: (row) =>
          row.auth ? (
            <StatusBadge tone="info">{t('apiDocs.authRequired')}</StatusBadge>
          ) : (
            <StatusBadge tone="neutral">{t('apiDocs.authPublic')}</StatusBadge>
          ),
        sortValue: (row) => (row.auth ? 1 : 0),
        className: 'w-28',
      },
      {
        id: 'desc',
        header: t('apiDocs.col.desc'),
        mobileRole: 'secondary',
        cell: (row) => (
          <span className="text-sm leading-relaxed text-muted-foreground">
            {t(row.descKey)}
          </span>
        ),
      },
      {
        id: 'copy',
        header: '',
        mobileRole: 'actions',
        cell: (row) => (
          <CopyButton
            value={`${origin}${row.path}`}
            label={t('common.copy')}
            size="icon-sm"
          />
        ),
        className: 'w-12',
        align: 'right',
      },
    ],
    [origin, t],
  )

  return (
    <Card className="min-w-0">
      <CardHeader className="gap-1.5">
        <CardTitle>{t('apiDocs.endpointsTitle')}</CardTitle>
        <CardDescription>{t('apiDocs.endpointsDesc')}</CardDescription>
      </CardHeader>
      <CardContent className="min-w-0">
        <DataTable
          rows={API_ENDPOINTS}
          columns={columns}
          rowKey={(r) => r.path}
          pageSize={20}
          emptyMessage={t('api.noModels')}
        />
      </CardContent>
    </Card>
  )
}
