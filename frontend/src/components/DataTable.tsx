import type { ReactNode } from 'react'

import { Card } from './Card'
import { ErrorBanner } from './ErrorBanner'
import { Spinner } from './Spinner'

/**
 * One table implementation for the whole app. Seven pages previously repeated
 * the same scaffolding (RTL-aware header row, divide-y body, hover state,
 * horizontal scroll wrapper, and the loading / error / empty branches); they
 * now describe their columns and hand over the rows.
 */
export interface Column<T> {
  /** Column heading (Persian). */
  header: ReactNode
  /** Cell renderer. */
  cell: (row: T) => ReactNode
  /** Optional extra classes for the cell (e.g. text-end for actions). */
  className?: string
}

interface DataTableProps<T> {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  /** Card heading; omit for a bare table. */
  title?: ReactNode
  /** Filter controls rendered above the table. */
  toolbar?: ReactNode
  /** Rendered under the table (typically <Pagination />). */
  footer?: ReactNode
  loading?: boolean
  error?: string | null
  /** Shown when there are no rows and no error. */
  emptyMessage?: string
  /** Highlight predicate, e.g. the currently-active coverage rule version. */
  rowClassName?: (row: T) => string | undefined
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  title,
  toolbar,
  footer,
  loading = false,
  error = null,
  emptyMessage = 'موردی یافت نشد.',
  rowClassName,
}: DataTableProps<T>) {
  return (
    <Card className="!p-0">
      {title && (
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">{title}</h2>
        </div>
      )}

      {toolbar && (
        <div className="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          {toolbar}
        </div>
      )}

      {error && (
        <div className="p-5">
          <ErrorBanner message={error} />
        </div>
      )}

      {loading ? (
        <Spinner />
      ) : rows.length === 0 ? (
        <p className="p-5 text-sm text-slate-500 dark:text-slate-400">{emptyMessage}</p>
      ) : (
        <div className="scroll-x">
          <table className="w-full text-start text-sm">
            <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
              <tr>
                {columns.map((col, i) => (
                  <th key={i} className={`px-5 py-2 font-medium ${col.className ?? ''}`}>
                    {col.header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {rows.map((row) => (
                <tr
                  key={rowKey(row)}
                  className={rowClassName?.(row) ?? 'hover:bg-slate-50 dark:hover:bg-slate-800/50'}
                >
                  {columns.map((col, i) => (
                    <td key={i} className={`px-5 py-3 ${col.className ?? ''}`}>
                      {col.cell(row)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {footer && <div className="px-5 pb-4">{footer}</div>}
    </Card>
  )
}
