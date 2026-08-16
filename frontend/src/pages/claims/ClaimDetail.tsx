import { useCallback, useEffect, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  approveClaim,
  closeClaim,
  getClaim,
  getClaimHistory,
  markPaid,
  rejectClaim,
  resubmitClaim,
  returnForDocs,
  startReview,
  submitClaim,
} from '../../api/claims'
import { apiErrorMessage } from '../../api/client'
import { getEmployee } from '../../api/employees'
import type { AuditLog, Claim, Employee } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { useServiceTypes } from '../../hooks/useServiceTypes'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { StatusBadge } from '../../components/StatusBadge'
import { JsonViewer } from '../../components/JsonViewer'
import { ClaimAttachments } from './ClaimAttachments'
import { BENEFICIARY_LABELS, auditActionLabel, formatDate, formatDateTime, formatMoney, formatNumber } from '../../lib/format'

type ReasonAction = 'reject' | 'return-for-docs'

export function ClaimDetail() {
  const { id } = useParams<{ id: string }>()
  const { user } = useAuth()
  const { showToast } = useToast()
  const navigate = useNavigate()
  const { byId: serviceTypeById } = useServiceTypes()

  const [claim, setClaim] = useState<Claim | null>(null)
  const [employee, setEmployee] = useState<Employee | null>(null)
  const [history, setHistory] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionPending, setActionPending] = useState(false)
  const [reasonAction, setReasonAction] = useState<ReasonAction | null>(null)
  const [reason, setReason] = useState('')

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const [claimData, historyData] = await Promise.all([getClaim(id), getClaimHistory(id)])
      setClaim(claimData)
      setHistory(historyData)
      try {
        const emp = await getEmployee(claimData.employee_id)
        setEmployee(emp)
      } catch {
        // Auditors and some roles cannot fetch employee records directly; not fatal.
        setEmployee(null)
      }
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    load()
  }, [load])

  async function runAction(fn: () => Promise<Claim>) {
    setActionPending(true)
    setError(null)
    try {
      const updated = await fn()
      setClaim(updated)
      const historyData = await getClaimHistory(updated.id)
      setHistory(historyData)
      showToast('درخواست به‌روزرسانی شد.', 'success')
    } catch (err) {
      const msg = apiErrorMessage(err)
      setError(msg)
      showToast(msg, 'error')
    } finally {
      setActionPending(false)
    }
  }

  function openReason(action: ReasonAction) {
    setReason('')
    setReasonAction(action)
  }

  async function confirmReason() {
    if (!claim || !reasonAction) return
    if (!reason.trim()) {
      showToast('وارد کردن دلیل الزامی است.', 'error')
      return
    }
    const action = reasonAction
    setReasonAction(null)
    if (action === 'reject') {
      await runAction(() => rejectClaim(claim.id, reason.trim()))
    } else {
      await runAction(() => returnForDocs(claim.id, reason.trim()))
    }
  }

  if (loading) return <Spinner />
  if (error && !claim) return <ErrorBanner message={error} />
  if (!claim) return null

  const isOwner = user?.role === 'employee' && user.employee_id === claim.employee_id
  const isReviewerLike = user?.role === 'reviewer' || user?.role === 'admin'
  const serviceType = serviceTypeById.get(claim.service_type_id)
  // Mirrors the server rule: the claim's owner (or an admin) may add documents
  // while the claim is a draft or was returned for missing paperwork.
  const canUploadDocs =
    (isOwner || user?.role === 'admin') &&
    (claim.status === 'draft' || claim.status === 'returned_for_docs')

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <button
            type="button"
            onClick={() => navigate(-1)}
            className="text-sm text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
          >
            → بازگشت
          </button>
          <h1 className="mt-1 text-xl font-semibold text-slate-900 dark:text-slate-50">
            درخواست <span className="dir-ltr">{claim.id.slice(0, 8)}</span>
          </h1>
        </div>
        <StatusBadge status={claim.status} />
      </div>

      {error && <ErrorBanner message={error} />}

      <Card>
        <dl className="grid grid-cols-2 gap-4 text-sm">
          <Detail
            label="کارمند"
            value={
              employee ? (
                <>
                  {employee.full_name} (<span className="dir-ltr">{employee.personnel_no}</span>)
                </>
              ) : (
                <span className="dir-ltr">{claim.employee_id}</span>
              )
            }
          />
          <Detail label="ذی‌نفع" value={BENEFICIARY_LABELS[claim.beneficiary_type]} />
          <Detail
            label="نوع خدمت"
            value={serviceType ? serviceType.name : <span className="dir-ltr">{claim.service_type_id}</span>}
          />
          <Detail label="تاریخ فاکتور" value={formatDate(claim.receipt_date)} />
          <Detail label="مبلغ درخواستی" value={formatMoney(claim.requested_amount)} />
          <Detail label="درصد پوشش اعمال‌شده" value={claim.coverage_percent_applied != null ? `${formatNumber(claim.coverage_percent_applied)}٪` : '—'} />
          <Detail label="مبلغ قابل پرداخت" value={formatMoney(claim.payable_amount)} />
          <Detail label="تاریخ ثبت" value={formatDateTime(claim.submitted_at)} />
          <Detail label="تاریخ بررسی" value={formatDateTime(claim.reviewed_at)} />
          <Detail label="تاریخ پرداخت" value={formatDateTime(claim.paid_at)} />
          <Detail label="تاریخ بستن" value={formatDateTime(claim.closed_at)} />
        </dl>
        {claim.description && (
          <div className="mt-4 border-t border-slate-100 pt-4 text-sm dark:border-slate-800">
            <div className="font-medium text-slate-500 dark:text-slate-400">توضیحات</div>
            <p className="mt-1 text-slate-800 dark:text-slate-200">{claim.description}</p>
          </div>
        )}
        {claim.rejection_reason && (
          <div className="mt-4 border-t border-slate-100 pt-4 text-sm dark:border-slate-800">
            <div className="font-medium text-slate-500 dark:text-slate-400">دلیل</div>
            <p className="mt-1 text-slate-800 dark:text-slate-200">{claim.rejection_reason}</p>
          </div>
        )}
      </Card>

      <ClaimAttachments
        claimId={claim.id}
        canUpload={canUploadDocs}
        onUploaded={() => {
          // The upload is audited, so the history needs refreshing too.
          getClaimHistory(claim.id).then(setHistory).catch(() => {})
        }}
      />

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">عملیات</h2>
        <div className="mt-3 flex flex-wrap gap-2">
          {isOwner && claim.status === 'draft' && (
            <ActionButton disabled={actionPending} onClick={() => runAction(() => submitClaim(claim.id))}>
              ثبت
            </ActionButton>
          )}
          {isOwner && claim.status === 'returned_for_docs' && (
            <ActionButton disabled={actionPending} onClick={() => runAction(() => resubmitClaim(claim.id))}>
              ارسال مجدد
            </ActionButton>
          )}
          {isReviewerLike && claim.status === 'submitted' && (
            <ActionButton disabled={actionPending} onClick={() => runAction(() => startReview(claim.id))}>
              شروع بررسی
            </ActionButton>
          )}
          {isReviewerLike && claim.status === 'under_review' && (
            <>
              <ActionButton disabled={actionPending} onClick={() => runAction(() => approveClaim(claim.id))}>
                تأیید
              </ActionButton>
              <ActionButton variant="danger" disabled={actionPending} onClick={() => openReason('reject')}>
                رد کردن
              </ActionButton>
              <ActionButton variant="secondary" disabled={actionPending} onClick={() => openReason('return-for-docs')}>
                بازگرداندن برای مدارک
              </ActionButton>
            </>
          )}
          {isReviewerLike && claim.status === 'approved' && (
            <ActionButton disabled={actionPending} onClick={() => runAction(() => markPaid(claim.id))}>
              ثبت پرداخت
            </ActionButton>
          )}
          {isReviewerLike && (claim.status === 'paid' || claim.status === 'rejected') && (
            <ActionButton disabled={actionPending} onClick={() => runAction(() => closeClaim(claim.id))}>
              بستن
            </ActionButton>
          )}
          {!isOwner && !isReviewerLike && (
            <p className="text-sm text-slate-400 dark:text-slate-500">عملیاتی برای نقش شما در دسترس نیست.</p>
          )}
        </div>

        {reasonAction && (
          <div className="mt-4 rounded-lg border border-slate-200 p-4 dark:border-slate-700">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
              دلیل {reasonAction === 'reject' ? 'رد کردن' : 'بازگرداندن برای مدارک'}
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={3}
                autoFocus
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
              />
            </label>
            <div className="mt-3 flex gap-2">
              <button
                type="button"
                onClick={confirmReason}
                className="rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-brand-700"
              >
                تأیید
              </button>
              <button
                type="button"
                onClick={() => setReasonAction(null)}
                className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-600 dark:border-slate-700 dark:text-slate-300"
              >
                انصراف
              </button>
            </div>
          </div>
        )}
      </Card>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">تاریخچه</h2>
          {(user?.role === 'admin' || user?.role === 'auditor') && id && (
            <Link
              to={`/audit-logs?entity_type=claim&entity_id=${encodeURIComponent(id)}`}
              className="text-xs font-medium text-brand-700 hover:underline dark:text-brand-300"
            >
              مشاهده در تاریخچهٔ کامل
            </Link>
          )}
        </div>
        {history.length === 0 ? (
          <p className="mt-2 text-sm text-slate-400 dark:text-slate-500">هنوز تاریخچه‌ای وجود ندارد.</p>
        ) : (
          <ol className="mt-3 space-y-4 border-s border-slate-200 ps-4 dark:border-slate-700">
            {history.map((entry) => (
              <li key={entry.id} className="relative">
                <span className="absolute -start-[21px] top-1 h-2.5 w-2.5 rounded-full bg-brand-500" />
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium text-slate-800 dark:text-slate-100">{auditActionLabel(entry.action)}</span>
                  <span className="text-xs text-slate-400 dark:text-slate-500">{formatDateTime(entry.occurred_at)}</span>
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  توسط {entry.actor_username ? <span className="dir-ltr">{entry.actor_username}</span> : 'سامانه'}
                </div>
                <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                  <JsonViewer data={entry.before_data} label="قبل" />
                  <JsonViewer data={entry.after_data} label="بعد" />
                </div>
              </li>
            ))}
          </ol>
        )}
      </Card>
    </div>
  )
}

function Detail({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500">{label}</dt>
      <dd className="mt-0.5 text-slate-800 dark:text-slate-100">{value}</dd>
    </div>
  )
}

function ActionButton({
  children,
  onClick,
  disabled,
  variant = 'primary',
}: {
  children: string
  onClick: () => void
  disabled?: boolean
  variant?: 'primary' | 'secondary' | 'danger'
}) {
  const styles = {
    primary: 'bg-brand-600 text-white hover:bg-brand-700',
    secondary: 'border border-slate-300 text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800',
    danger: 'bg-red-600 text-white hover:bg-red-700',
  }[variant]

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`rounded-lg px-4 py-2 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60 ${styles}`}
    >
      {children}
    </button>
  )
}

// Re-export so App.tsx can link to claim detail from the audit log / elsewhere.
export function claimDetailPath(id: string) {
  return `/claims/${id}`
}

// Used by other pages that want a plain text link to a claim.
export function ClaimLink({ id }: { id: string }) {
  return (
    <Link to={claimDetailPath(id)} className="text-brand-600 hover:underline">
      {id.slice(0, 8)}
    </Link>
  )
}
