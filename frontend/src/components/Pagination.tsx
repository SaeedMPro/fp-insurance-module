import { formatNumber } from '../lib/format'

interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, pageSize, total, onPageChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-between border-t border-slate-100 pt-3 text-sm text-slate-500 dark:border-slate-800 dark:text-slate-400">
      <span>
        صفحه {formatNumber(page)} از {formatNumber(totalPages)} · {formatNumber(total)} مورد
      </span>
      <div className="flex gap-2">
        <button
          type="button"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          className="rounded-md border border-slate-200 px-3 py-1 font-medium disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-700"
        >
          قبلی
        </button>
        <button
          type="button"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
          className="rounded-md border border-slate-200 px-3 py-1 font-medium disabled:cursor-not-allowed disabled:opacity-40 dark:border-slate-700"
        >
          بعدی
        </button>
      </div>
    </div>
  )
}
