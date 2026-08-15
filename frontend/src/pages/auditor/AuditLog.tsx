import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { listAuditLogs } from '../../api/auditLogs'
import type { ListAuditLogsParams } from '../../api/auditLogs'
import { apiErrorMessage } from '../../api/client'
import type { AuditLog as AuditLogEntry } from '../../api/types'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Pagination } from '../../components/Pagination'
import { JsonViewer } from '../../components/JsonViewer'
import { PersianDateInput } from '../../components/PersianDateInput'
import {
  formatDateTime,
  dateInputToRFC3339,
  auditActionLabel,
  entityTypeLabel,
  AUDIT_ACTION_LABELS,
} from '../../lib/format'

const PAGE_SIZE = 25

const SUBJECT_TYPES = ['', 'claim', 'coverage_rule', 'user'] as const

const ACTION_OPTIONS = Object.entries(AUDIT_ACTION_LABELS)

const filterControlClass =
  'mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100'

type Focus = { entityType: string; entityId: string }

function shortId(id: string): string {
  if (id.length <= 12) return id
  return `${id.slice(0, 8)}…`
}

function entityHref(entityType: string, entityId: string): string | null {
  if (entityType === 'claim') return `/claims/${entityId}`
  if (entityType === 'coverage_rule') return '/coverage-rules'
  if (entityType === 'user') return '/users'
  return null
}

function initialFromSearch(searchParams: URLSearchParams): ListAuditLogsParams {
  return {
    entity_type: searchParams.get('entity_type') || undefined,
    entity_id: searchParams.get('entity_id') || undefined,
    action: searchParams.get('action') || undefined,
    // No default date range — show the full history until the user filters.
  }
}

export function AuditLog() {
  const [searchParams, setSearchParams] = useSearchParams()

  const [entityType, setEntityType] = useState(() => searchParams.get('entity_type') ?? '')
  const [action, setAction] = useState(() => searchParams.get('action') ?? '')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [focus, setFocus] = useState<Focus | null>(() => {
    const id = searchParams.get('entity_id')
    const type = searchParams.get('entity_type')
    return id && type ? { entityType: type, entityId: id } : null
  })

  const [applied, setApplied] = useState<ListAuditLogsParams>(() => initialFromSearch(searchParams))

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

  function syncUrl(next: { entityType?: string; entityId?: string; action?: string }) {
    const params = new URLSearchParams()
    if (next.entityType) params.set('entity_type', next.entityType)
    if (next.entityId) params.set('entity_id', next.entityId)
    if (next.action) params.set('action', next.action)
    setSearchParams(params, { replace: true })
  }

  function applyFilters() {
    setPage(1)
    const nextFocus = focus
    setApplied({
      entity_type: nextFocus?.entityType || entityType || undefined,
      entity_id: nextFocus?.entityId || undefined,
      action: action || undefined,
      from: from ? dateInputToRFC3339(from) : undefined,
      to: to ? dateInputToRFC3339(to) : undefined,
    })
    syncUrl({
      entityType: nextFocus?.entityType || entityType || undefined,
      entityId: nextFocus?.entityId,
      action: action || undefined,
    })
  }

  function clearFilters() {
    setEntityType('')
    setAction('')
    setFocus(null)
    setFrom('')
    setTo('')
    setPage(1)
    setApplied({})
    setSearchParams({}, { replace: true })
  }

  function focusOn(row: AuditLogEntry) {
    const next: Focus = { entityType: row.entity_type, entityId: row.entity_id }
    setFocus(next)
    setEntityType(row.entity_type)
    setPage(1)
    setApplied((prev) => ({
      ...prev,
      entity_type: next.entityType,
      entity_id: next.entityId,
    }))
    syncUrl({ entityType: next.entityType, entityId: next.entityId, action: action || undefined })
  }

  function clearFocus() {
    setFocus(null)
    setPage(1)
    setApplied((prev) => ({
      ...prev,
      entity_type: entityType || undefined,
      entity_id: undefined,
    }))
    syncUrl({ entityType: entityType || undefined, action: action || undefined })
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">تاریخچه اقدامات</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          ببینید چه کسی، چه کاری، روی چه چیزی انجام داده است — همراه با زمان و وضعیت قبل/بعد.
        </p>
      </div>

      <Card>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label className="text-sm font-medium text-slate-600 dark:text-slate-300">
            موضوع
            <select
              value={focus ? focus.entityType : entityType}
              disabled={!!focus}
              onChange={(e) => setEntityType(e.target.value)}
              className={filterControlClass}
            >
              {SUBJECT_TYPES.map((t) => (
                <option key={t || 'all'} value={t}>
                  {t === '' ? 'همه موضوعات' : entityTypeLabel(t)}
                </option>
              ))}
            </select>
          </label>

          <label className="text-sm font-medium text-slate-600 dark:text-slate-300">
            نوع اقدام
            <select value={action} onChange={(e) => setAction(e.target.value)} className={filterControlClass}>
              <option value="">همه اقدامات</option>
              {ACTION_OPTIONS.map(([code, label]) => (
                <option key={code} value={code}>
                  {label}
                </option>
              ))}
            </select>
          </label>

          <div className="hidden lg:block" />

          <label className="text-sm font-medium text-slate-600 dark:text-slate-300">
            از تاریخ
            <PersianDateInput value={from} onChange={setFrom} className={filterControlClass} />
          </label>

          <label className="text-sm font-medium text-slate-600 dark:text-slate-300">
            تا تاریخ
            <PersianDateInput value={to} onChange={setTo} className={filterControlClass} />
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

        {focus && (
          <div className="mt-4 flex flex-wrap items-center gap-2 rounded-lg border border-brand-200 bg-brand-50 px-3 py-2 text-sm text-brand-900 dark:border-brand-800 dark:bg-brand-950 dark:text-brand-100">
            <span>
              فقط اقدامات مربوط به این {entityTypeLabel(focus.entityType)}
              <span className="dir-ltr ms-2 font-mono text-xs opacity-70">{shortId(focus.entityId)}</span>
            </span>
            {entityHref(focus.entityType, focus.entityId) && (
              <Link
                to={entityHref(focus.entityType, focus.entityId)!}
                className="font-medium text-brand-700 underline-offset-2 hover:underline dark:text-brand-300"
              >
                باز کردن
              </Link>
            )}
            <button
              type="button"
              onClick={clearFocus}
              className="ms-auto rounded-md px-2 py-0.5 text-xs font-medium text-brand-800 hover:bg-brand-100 dark:text-brand-200 dark:hover:bg-brand-900"
            >
              برداشتن این فیلتر
            </button>
          </div>
        )}
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
          {rows.map((row) => {
            const href = entityHref(row.entity_type, row.entity_id)
            return (
              <Card key={row.id}>
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="inline-flex items-center rounded-full bg-brand-100 px-2.5 py-0.5 text-xs font-semibold text-brand-800 dark:bg-brand-900 dark:text-brand-200">
                      {auditActionLabel(row.action)}
                    </span>
                    <span className="text-sm font-medium text-slate-900 dark:text-slate-100">
                      {entityTypeLabel(row.entity_type)}
                    </span>
                    {href ? (
                      <Link
                        to={href}
                        className="text-sm text-brand-700 hover:underline dark:text-brand-300"
                        title={row.entity_id}
                      >
                        مشاهده
                      </Link>
                    ) : null}
                    {!focus && (
                      <button
                        type="button"
                        onClick={() => focusOn(row)}
                        className="text-xs text-slate-500 underline-offset-2 hover:text-slate-800 hover:underline dark:hover:text-slate-200"
                      >
                        فقط همین مورد
                      </button>
                    )}
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
            )
          })}
          <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPageChange={setPage} />
        </div>
      )}
    </div>
  )
}
