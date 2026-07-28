// DataTable — shared client-side table with sort + pagination. Used by API Keys
// and the Dashboard Usage table. Search/filter is done by the caller (it owns
// the domain-specific predicate); this component takes the already-filtered rows
// and handles column sort + paging + the empty state.
//
// Columns declare an optional `sortValue` to enable header-click sorting; a
// column without it renders but isn't sortable.
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
import { tp } from '@/lib/t'
import { cn } from '@/lib/utils'

export interface Column<T> {
  id: string
  header: ReactNode
  cell: (row: T) => ReactNode
  sortValue?: (row: T) => string | number
  className?: string
  align?: 'left' | 'right' | 'center'
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

      <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
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

        <div className="flex items-center gap-2">
          {footer}
          <span>{tp(t, 'apiKeys.pageOf', clampedPage + 1, pageCount)}</span>
          <Button
            variant="outline"
            size="sm"
            disabled={clampedPage <= 0}
            onClick={() => setPage(clampedPage - 1)}
          >
            {t('apiKeys.prev')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={clampedPage >= pageCount - 1}
            onClick={() => setPage(clampedPage + 1)}
          >
            {t('apiKeys.next')}
          </Button>
        </div>
      </div>
    </div>
  )
}
