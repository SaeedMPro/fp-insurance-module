import type { BeneficiaryType, ClaimStatus, Relation, Role } from '../api/types'

// The UI is Persian (fa-IR): numbers use Persian digits with grouping, and dates
// render in the Jalali calendar. Values are stored/entered as Gregorian and
// converted for display here.

const faNumber = new Intl.NumberFormat('fa-IR', { maximumFractionDigits: 0 })

/** Format a Rial amount with Persian digits and thousands separators. */
export function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  return faNumber.format(value)
}

/** Format any integer-ish value with Persian digits (no currency semantics). */
export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  return faNumber.format(value)
}

const faDateTime = new Intl.DateTimeFormat('fa-IR', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

const faDate = new Intl.DateTimeFormat('fa-IR', {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
})

/** Format an ISO/RFC3339 timestamp as a Jalali date + time for display. */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return faDateTime.format(d)
}

/** Format an ISO/RFC3339 timestamp as a Jalali date for display. */
export function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return faDate.format(d)
}

/** Convert a bare <input type="date"> value ("YYYY-MM-DD", Gregorian) to RFC3339 midnight UTC. */
export function dateInputToRFC3339(value: string): string {
  if (!value) return value
  return `${value}T00:00:00Z`
}

/** Convert an RFC3339 timestamp to a bare "YYYY-MM-DD" for <input type="date">. */
export function rfc3339ToDateInput(value: string | null | undefined): string {
  if (!value) return ''
  return value.slice(0, 10)
}

// ---- Enum → Persian label maps (single source of truth for display text) ----

export const CLAIM_STATUS_LABELS: Record<ClaimStatus, string> = {
  draft: 'پیش‌نویس',
  submitted: 'ثبت‌شده',
  under_review: 'در حال بررسی',
  returned_for_docs: 'بازگشت برای تکمیل مدارک',
  approved: 'تأییدشده',
  rejected: 'رد شده',
  payment_calculated: 'مبلغ محاسبه‌شده',
  paid: 'پرداخت‌شده',
  closed: 'بسته‌شده',
}

export const CLAIM_STATUS_COLORS: Record<ClaimStatus, string> = {
  draft: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
  submitted: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300',
  under_review: 'bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300',
  returned_for_docs: 'bg-orange-100 text-orange-800 dark:bg-orange-950 dark:text-orange-300',
  approved: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300',
  rejected: 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300',
  payment_calculated: 'bg-teal-100 text-teal-800 dark:bg-teal-950 dark:text-teal-300',
  paid: 'bg-indigo-100 text-indigo-800 dark:bg-indigo-950 dark:text-indigo-300',
  closed: 'bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
}

export const ROLE_LABELS: Record<Role, string> = {
  admin: 'مدیر سامانه',
  reviewer: 'کارشناس بررسی',
  employee: 'کارمند',
  auditor: 'بازرس',
}

export const RELATION_LABELS: Record<Relation, string> = {
  self: 'شخص بیمه‌شده',
  spouse: 'همسر',
  child: 'فرزند',
  parent: 'والدین',
}

export const BENEFICIARY_LABELS: Record<BeneficiaryType, string> = {
  self: 'شخص بیمه‌شده',
  dependent: 'عضو تحت تکفل',
}

export const EMPLOYMENT_STATUS_LABELS: Record<string, string> = {
  active: 'شاغل',
  terminated: 'پایان همکاری',
}

// Audit-trail vocabulary. These map the backend's English action/entity names
// (see internal/workflow and internal/audit) to Persian for display. Unknown
// values fall back to the raw token via the helper below.
export const AUDIT_ACTION_LABELS: Record<string, string> = {
  submit: 'ثبت',
  resubmit: 'ارسال مجدد',
  start_review: 'شروع بررسی',
  approve: 'تأیید',
  reject: 'رد',
  return_for_docs: 'بازگرداندن برای مدارک',
  mark_paid: 'ثبت پرداخت',
  close: 'بستن',
  config_change: 'تغییر پیکربندی',
  login: 'ورود',
}

export const ENTITY_TYPE_LABELS: Record<string, string> = {
  claim: 'درخواست',
  coverage_rule: 'قانون پوشش',
  user: 'کاربر',
}

export function auditActionLabel(action: string): string {
  return AUDIT_ACTION_LABELS[action] ?? action
}

export function entityTypeLabel(entityType: string): string {
  return ENTITY_TYPE_LABELS[entityType] ?? entityType
}
