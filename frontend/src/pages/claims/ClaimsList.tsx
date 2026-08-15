import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listClaims } from '../../api/claims'
import { apiErrorMessage } from '../../api/client'
import type { Claim, ClaimStatus } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { useServiceTypes } from '../../hooks/useServiceTypes'
import { StatusBadge } from '../../components/StatusBadge'
import { Spinner } from '../../components/Spinner'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Pagination } from '../../components/Pagination'
import { Card } from '../../components/Card'
import { CLAIM_STATUS_LABELS, formatDate, formatMoney } from '../../lib/format'

const PAGE_SIZE = 20

const STATUS_OPTIONS: { value: ClaimStatus | ''; label: string }[] = [
  { value: '', label: 'همه وضعیت‌ها' },
  { value: 'draft', label: CLAIM_STATUS_LABELS.draft },
  { value: 'submitted', label: CLAIM_STATUS_LABELS.submitted },
  { value: 'under_review', label: CLAIM_STATUS_LABELS.under_review },
  { value: 'returned_for_docs', label: CLAIM_STATUS_LABELS.returned_for_docs },
  { value: 'approved', label: CLAIM_STATUS_LABELS.approved },
  { value: 'rejected', label: CLAIM_STATUS_LABELS.rejected },
  { value: 'paid', label: CLAIM_STATUS_LABELS.paid },
  { value: 'closed', label: CLAIM_STATUS_LABELS.closed },
]

/** Shared claims list: "My Claims" for employees, "Review Queue" for reviewers, "All Claims" for admins. */
export function ClaimsList() {
  const { user } = useAuth()
  const { byId: serviceTypeById } = useServiceTypes()
  const isEmployee = user?.role === 'employee'
  const isReviewerLike = user?.role === 'reviewer' || user?.role === 'admin'

  const [status, setStatus] = useState<ClaimStatus | ''>(isReviewerLike ? '' : '')
  const [reviewQueueOnly, setReviewQueueOnly] = useState(user?.role === 'reviewer')
  const [claims, setClaims] = useState<Claim[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    listClaims({
      status: status || undefined,
      page,
      page_size: PAGE_SIZE,
    })
      .then((res) => {
        if (cancelled) return
        let items = res.items
        if (reviewQueueOnly && !status) {
          items = items.filter((c) => c.status === 'submitted' || c.status === 'under_review')
        }
        setClaims(items)
        setTotal(res.total)
      })
      .catch((err) => !cancelled && setError(apiErrorMessage(err)))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [status, page, reviewQueueOnly])

  const title = isEmployee ? 'درخواست‌های من' : user?.role === 'reviewer' ? 'کارتابل بررسی' : 'همه درخواست‌ها'

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">{title}</h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            {isEmployee
              ? 'درخواست‌هایی که برای بازپرداخت ثبت کرده‌اید.'
              : 'همهٔ درخواست‌های قابل مشاهده برای نقش شما.'}
          </p>
        </div>
        {isEmployee && (
          <Link
            to="/claims/new"
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            ثبت درخواست جدید
          </Link>
        )}
      </div>

      <Card className="!p-0">
        <div className="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          <label className="text-sm text-slate-500 dark:text-slate-400">
            وضعیت
            <select
              value={status}
              onChange={(e) => {
                setStatus(e.target.value as ClaimStatus | '')
                setPage(1)
              }}
              className="ms-2 rounded-md border border-slate-300 bg-white px-2 py-1 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </label>
          {user?.role === 'reviewer' && (
            <label className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
              <input
                type="checkbox"
                checked={reviewQueueOnly}
                onChange={(e) => setReviewQueueOnly(e.target.checked)}
                disabled={!!status}
                className="rounded border-slate-300"
              />
              فقط ثبت‌شده / در حال بررسی
            </label>
          )}
        </div>

        {error && (
          <div className="p-5">
            <ErrorBanner message={error} />
          </div>
        )}
        {loading ? (
          <Spinner />
        ) : claims.length === 0 ? (
          <p className="p-5 text-sm text-slate-500 dark:text-slate-400">درخواستی یافت نشد.</p>
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">تاریخ فاکتور</th>
                  <th className="px-5 py-2 font-medium">نوع خدمت</th>
                  <th className="px-5 py-2 font-medium">مبلغ درخواستی</th>
                  <th className="px-5 py-2 font-medium">مبلغ قابل پرداخت</th>
                  <th className="px-5 py-2 font-medium">وضعیت</th>
                  <th className="px-5 py-2 font-medium" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {claims.map((claim) => {
                  const st = serviceTypeById.get(claim.service_type_id)
                  return (
                    <tr key={claim.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                      <td className="px-5 py-3">{formatDate(claim.receipt_date)}</td>
                      <td className="px-5 py-3">{st ? st.name : '—'}</td>
                      <td className="px-5 py-3">{formatMoney(claim.requested_amount)}</td>
                      <td className="px-5 py-3">{formatMoney(claim.payable_amount)}</td>
                      <td className="px-5 py-3">
                        <StatusBadge status={claim.status} />
                      </td>
                      <td className="px-5 py-3 text-end">
                        <Link
                          to={`/claims/${claim.id}`}
                          className="text-sm font-medium text-brand-600 hover:underline"
                        >
                          مشاهده
                        </Link>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}

        <div className="px-5 pb-4">
          <Pagination page={page} pageSize={PAGE_SIZE} total={total} onPageChange={setPage} />
        </div>
      </Card>
    </div>
  )
}
