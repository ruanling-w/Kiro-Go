// DataTable — shared client-side table with sort + pagination. Used by API Keys,
// Dashboard Usage, and API docs endpoints. Search/filter is done by the caller.
//
// Desktop (md+): classic table with header-click sort.
// Mobile (<md): card stack driven by column `mobileRole` metadata + optional
// compact "Sort by" select for sortable columns.
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { EmptyState } from './EmptyState'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { tp } from '@/lib/t'
import { cn } from '@/lib/utils'

export interface Column<T> {
  id: string
  header: ReactNode
  cell: (row: T) => ReactNode
  sortValue?: (row: T) => string | number
  className?: string
  align?: 'left' | 'right' | 'center'
  /** Skip this column entirely in the mobile card body. */
  mobileHidden?: boolean
  /**
   * Mobile card layout role:
   * - primary: title (left of header row)
   * - badge: sits opposite primary (e.g. status pill)
   * - secondary: full-width under title (e.g. key reveal)
   * - meta: stacked label | value rows
   * - actions: bottom action row
   * Default: first non-actions col → primary, rest → meta (actions id heuristic).
   */
  mobileRole?: 'primary' | 'badge' | 'secondary' | 'meta' | 'actions'
}

interface DataTableProps<T> {
  rows: T[]
  columns: Column<T>[]
  rowKey: (row: T) => string
  pageSize?: number
  pageSizeOptions?: number[]
  emptyMessage?: string
  footer?: ReactNode
  initialSort?: { id: string; dir: 'asc' | 'desc' }
}

const DEFAULT_PAGE_SIZES = [10, 20, 50, 100]

function headerText(header: ReactNode): string {
  if (typeof header === 'string' || typeof header === 'number') return String(header)
  return ''
}

function resolveMobileRole<T>(col: Column<T>, index: number, columns: Column<T>[]): NonNullable<Column<T>['mobileRole']> {
  if (col.mobileRole) return col.mobileRole
  if (col.id === 'actions') return 'actions'
  const firstContentIdx = columns.findIndex((c) => c.id !== 'actions' && !c.mobileHidden)
  if (index === firstContentIdx) return 'primary'
  return 'meta'
}

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  pageSize: initialPageSize = 20,
  pageSizeOptions = DEFAULT_PAGE_SIZES,
  emptyMessage,
  footer,
  initialSort,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  const [sort, setSort] = useState<{ id: string; dir: 'asc' | 'desc' } | null>(
    initialSort ?? null,
  )
  const [pageSize, setPageSize] = useState(initialPageSize)
  const [page, setPage] = useState(0)

  const sorted = useMemo(() => {
    if (!sort) return rows
    const col = columns.find((c) => c.id === sort.id)
    if (!col?.sortValue) return rows
    const factor = sort.dir === 'asc' ? 1 : -1
    return [...rows].sort((a, b) => {
      const av = col.sortValue!(a)
      const bv = col.sortValue!(b)
      if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * factor
      return String(av).localeCompare(String(bv)) * factor
    })
  }, [rows, sort, columns])

  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize))
  const clampedPage = Math.min(page, pageCount - 1)
  const start = clampedPage * pageSize
  const pageRows = sorted.slice(start, start + pageSize)

  const sortableCols = useMemo(
    () => columns.filter((c) => !!c.sortValue && !c.mobileHidden),
    [columns],
  )

  function toggleSort(id: string) {
    setSort((prev) => {
      if (prev?.id !== id) return { id, dir: 'asc' }
      if (prev.dir === 'asc') return { id, dir: 'desc' }
      return null
    })
  }

  if (rows.length === 0) {
    return <EmptyState message={emptyMessage ?? t('apiKeys.noMatches')} />
  }

  const pagination = (
    <div className="flex flex-col gap-3 text-sm text-muted-foreground sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
      <div className="flex flex-wrap items-center gap-2">
        <span>{t('apiKeys.pageSize')}</span>
        <Select
          value={String(pageSize)}
          onValueChange={(v) => {
            setPageSize(Number(v))
            setPage(0)
          }}
        >
          <SelectTrigger className="h-7 w-20" size="sm">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map((n) => (
              <SelectItem key={n} value={String(n)}>
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span>
          {tp(t, 'apiKeys.showing', start + 1, Math.min(start + pageSize, sorted.length), sorted.length)}
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {footer}
        <span className="mr-auto sm:mr-0">{tp(t, 'apiKeys.pageOf', clampedPage + 1, pageCount)}</span>
        <Button
          variant="outline"
          size="sm"
          className="min-w-16 flex-1 sm:flex-none"
          disabled={clampedPage <= 0}
          onClick={() => setPage(clampedPage - 1)}
        >
          {t('apiKeys.prev')}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="min-w-16 flex-1 sm:flex-none"
          disabled={clampedPage >= pageCount - 1}
          onClick={() => setPage(clampedPage + 1)}
        >
          {t('apiKeys.next')}
        </Button>
      </div>
    </div>
  )

  if (isMobile) {
    return (
      <div className="space-y-3">
        {sortableCols.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="shrink-0 text-sm text-muted-foreground">{t('common.sortBy')}</span>
            <Select
              value={sort ? `${sort.id}:${sort.dir}` : 'none'}
              onValueChange={(v) => {
                if (v === 'none') {
                  setSort(null)
                  return
                }
                const [id, dir] = v.split(':') as [string, 'asc' | 'desc']
                setSort({ id, dir })
                setPage(0)
              }}
            >
              <SelectTrigger className="h-8 min-w-0 flex-1" size="sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">{t('common.sortDefault')}</SelectItem>
                {sortableCols.flatMap((col) => {
                  const label = headerText(col.header) || col.id
                  return [
                    <SelectItem key={`${col.id}:asc`} value={`${col.id}:asc`}>
                      {label} ↑
                    </SelectItem>,
                    <SelectItem key={`${col.id}:desc`} value={`${col.id}:desc`}>
                      {label} ↓
                    </SelectItem>,
                  ]
                })}
              </SelectContent>
            </Select>
          </div>
        )}

        <div className="space-y-2">
          {pageRows.map((row) => {
            const roles = columns.map((col, i) => ({
              col,
              role: resolveMobileRole(col, i, columns),
            }))
            const primary = roles.find((r) => r.role === 'primary' && !r.col.mobileHidden)
            const badges = roles.filter((r) => r.role === 'badge' && !r.col.mobileHidden)
            const secondary = roles.filter((r) => r.role === 'secondary' && !r.col.mobileHidden)
            const meta = roles.filter((r) => r.role === 'meta' && !r.col.mobileHidden)
            const actions = roles.filter((r) => r.role === 'actions' && !r.col.mobileHidden)

            return (
              <div
                key={rowKey(row)}
                className="space-y-3 rounded-lg border bg-card p-3 text-card-foreground"
              >
                {(primary || badges.length > 0) && (
                  <div className="flex items-start justify-between gap-3">
                    {primary ? (
                      <div className="min-w-0 flex-1 leading-snug">
                        {primary.col.cell(row)}
                      </div>
                    ) : (
                      <span />
                    )}
                    {badges.length > 0 && (
                      <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                        {badges.map(({ col }) => (
                          <div key={col.id}>{col.cell(row)}</div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {secondary.map(({ col }) => (
                  <div key={col.id} className="min-w-0">
                    {col.cell(row)}
                  </div>
                ))}

                {meta.length > 0 && (
                  <dl className="space-y-2 border-t pt-2.5 text-sm">
                    {meta.map(({ col }) => (
                      <div
                        key={col.id}
                        className="flex items-center justify-between gap-3"
                      >
                        <dt className="shrink-0 text-muted-foreground">
                          {headerText(col.header) || col.id}
                        </dt>
                        <dd className="min-w-0 text-right tabular-nums">
                          {col.cell(row)}
                        </dd>
                      </div>
                    ))}
                  </dl>
                )}

                {actions.length > 0 && (
                  <div className="flex flex-wrap items-center justify-end gap-1 border-t pt-2.5">
                    {actions.map(({ col }) => (
                      <div key={col.id} className="contents">
                        {col.cell(row)}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>

        {pagination}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="overflow-hidden rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((col) => {
                const sortable = !!col.sortValue
                const active = sort?.id === col.id
                return (
                  <TableHead
                    key={col.id}
                    className={cn(
                      col.align === 'right' && 'text-right',
                      col.align === 'center' && 'text-center',
                      col.className,
                    )}
                  >
                    {sortable ? (
                      <button
                        type="button"
                        onClick={() => toggleSort(col.id)}
                        className={cn(
                          'inline-flex items-center gap-1 hover:text-foreground',
                          col.align === 'right' && 'flex-row-reverse',
                        )}
                      >
                        {col.header}
                        {active ? (
                          sort!.dir === 'asc' ? (
                            <ArrowUp className="size-3.5" />
                          ) : (
                            <ArrowDown className="size-3.5" />
                          )
                        ) : (
                          <ArrowUpDown className="size-3.5 opacity-40" />
                        )}
                      </button>
                    ) : (
                      col.header
                    )}
                  </TableHead>
                )
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {pageRows.map((row) => (
              <TableRow key={rowKey(row)}>
                {columns.map((col) => (
                  <TableCell
                    key={col.id}
                    className={cn(
                      col.align === 'right' && 'text-right',
                      col.align === 'center' && 'text-center',
                      col.className,
                    )}
                  >
                    {col.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {pagination}
    </div>
  )
}
