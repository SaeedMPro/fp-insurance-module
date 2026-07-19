import { useEffect, useState } from 'react'
import { listAuditLogs } from '../../api/auditLogs'
import type { ListAuditLogsParams } from '../../api/auditLogs'
import { apiErrorMessage } from '../../api/client'
import type { AuditLog as AuditLogEntry } from '../../api/types'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Pagination } from '../../components/Pagination'
import { JsonViewer } from '../../components/JsonViewer'
import { formatDateTime, dateInputToRFC3339, auditActionLabel, entityTypeLabel } from '../../lib/format'

const PAGE_SIZE = 25

const ENTITY_TYPES = ['', 'claim', 'coverage_rule', 'user']

export function AuditLog() {
  const [entityType, setEntityType] = useState('')
  const [entityId, setEntityId] = useState('')
  const [action, setAction] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [applied, setApplied] = useState<ListAuditLogsParams>({})

  const [rows, setRows] = useState<AuditLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    listAuditLogs({ ...applied, page, page_size: PAGE_SIZE })
      .then((res) => {
        if (cancelled) return
        setRows(res.items)
        setTotal(res.total)
      })
      .catch((err) => !cancelled && setError(apiErrorMessage(err)))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [applied, page])

  function applyFilters() {
    setPage(1)
    setApplied({
      entity_type: entityType || undefined,
      entity_id: entityId || undefined,
      action: action || undefined,
      from: from ? dateInputToRFC3339(from) : undefined,
      to: to ? dateInputToRFC3339(to) : undefined,
    })
  }

  function clearFilters() {
    setEntityType('')
    setEntityId('')
    setAction('')
    setFrom('')
    setTo('')
    setPage(1)
    setApplied({})
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">تاریخچه اقدامات</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          هر اقدام تغییردهندهٔ وضعیت در سامانه، همراه با کاربر، زمان و تصویر قبل/بعد.
        </p>
      </div>

      <Card>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <label className="text-sm text-slate-500 dark:text-slate-400">
            نوع موجودیت
            <select
              value={entityType}
              onChange={(e) => setEntityType(e.target.value)}
              className="mt-1 block w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              {ENTITY_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t === '' ? 'همه' : entityTypeLabel(t)}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm text-slate-500 dark:text-slate-400">
            شناسهٔ موجودیت
            <input
              value={entityId}
              onChange={(e) => setEntityId(e.target.value)}
              placeholder="شناسه"
              className="mt-1 block w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <label className="text-sm text-slate-500 dark:text-slate-400">
            اقدام
            <input
              value={action}
              onChange={(e) => setAction(e.target.value)}
              placeholder="مثلاً approve، reject، config_change"
              className="mt-1 block w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <label className="text-sm text-slate-500 dark:text-slate-400">
            از
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="mt-1 block w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <label className="text-sm text-slate-500 dark:text-slate-400">
            تا
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="mt-1 block w-full rounded-md border border-slate-300 px-2 py-1.5 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <div className="flex items-end gap-2">
            <button
              type="button"
              onClick={applyFilters}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
            >
              اعمال
            </button>
            <button
              type="button"
              onClick={clearFilters}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              پاک کردن
            </button>
          </div>
        </div>
      </Card>

      {error && <ErrorBanner message={error} />}

      {loading ? (
        <Spinner />
      ) : rows.length === 0 ? (
        <Card>
          <p className="text-sm text-slate-500 dark:text-slate-400">موردی با این فیلترها یافت نشد.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {rows.map((row) => (
            <Card key={row.id}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <span className="inline-flex items-center rounded-full bg-brand-100 px-2.5 py-0.5 text-xs font-semibold text-brand-800 dark:bg-brand-900 dark:text-brand-200">
                    {auditActionLabel(row.action)}
                  </span>
                  <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
                    {entityTypeLabel(row.entity_type)}
                  </span>
                  <span className="dir-ltr font-mono text-xs text-slate-400 dark:text-slate-500">{row.entity_id}</span>
                </div>
                <div className="text-xs text-slate-400 dark:text-slate-500">
                  {row.actor_username || 'سامانه'} · {formatDateTime(row.occurred_at)}
                </div>
              </div>
              <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
                <JsonViewer data={row.before_data} label="قبل" />
                <JsonViewer data={row.after_data} label="بعد" />
              </div>
            </Card>
          ))}
          <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPageChange={setPage} />
        </div>
      )}
    </div>
  )
}
