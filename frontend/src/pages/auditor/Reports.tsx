import { useEffect, useState } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { getSpendByEmployee, getSpendByMonth, getSpendByServiceType, getSummary } from '../../api/reports'
import type { ReportDateRange } from '../../api/reports'
import { apiErrorMessage } from '../../api/client'
import type { ReportSummary, SpendByEmployee, SpendByMonth, SpendByServiceType } from '../../api/types'
import { Card, StatTile } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { formatMoney, formatNumber, dateInputToRFC3339 } from '../../lib/format'

const BAR_COLORS = ['#2563eb', '#0891b2', '#7c3aed', '#db2777', '#ea580c', '#16a34a', '#ca8a04']

function chartMoney(value: number): string {
  return formatMoney(value)
}

export function Reports() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')

  const [summary, setSummary] = useState<ReportSummary | null>(null)
  const [byEmployee, setByEmployee] = useState<SpendByEmployee[]>([])
  const [byServiceType, setByServiceType] = useState<SpendByServiceType[]>([])
  const [byMonth, setByMonth] = useState<SpendByMonth[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  function load(range: ReportDateRange) {
    setLoading(true)
    setError(null)
    Promise.all([
      getSummary(range),
      getSpendByEmployee(range),
      getSpendByServiceType(range),
      getSpendByMonth(range),
    ])
      .then(([s, e, st, m]) => {
        setSummary(s)
        setByEmployee(e)
        setByServiceType(st)
        setByMonth(m)
      })
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load({})
  }, [])

  function applyFilter() {
    load({
      from: from ? dateInputToRFC3339(from) : undefined,
      to: to ? dateInputToRFC3339(to) : undefined,
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">گزارش‌ها</h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
            هزینه‌ها و فعالیت درخواست‌ها در کل سازمان. مبالغ به ریال است.
          </p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <label className="text-sm text-slate-500 dark:text-slate-400">
            از
            <input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="mt-1 block rounded-md border border-slate-300 px-2 py-1 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <label className="text-sm text-slate-500 dark:text-slate-400">
            تا
            <input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="mt-1 block rounded-md border border-slate-300 px-2 py-1 text-sm text-slate-900 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
            />
          </label>
          <button
            type="button"
            onClick={applyFilter}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            اعمال
          </button>
        </div>
      </div>

      {error && <ErrorBanner message={error} />}

      {loading ? (
        <Spinner />
      ) : (
        <>
          {summary && (
            <div className="grid grid-cols-2 gap-4 lg:grid-cols-5">
              <StatTile label="کل درخواست‌ها" value={formatNumber(summary.total_claims)} />
              <StatTile label="کل پرداختی" value={formatMoney(summary.total_paid_amount)} hint="ریال" />
              <StatTile label="در انتظار بررسی" value={formatNumber(summary.pending_review)} />
              <StatTile label="در انتظار پرداخت" value={formatNumber(summary.approved_awaiting_payment)} />
              <StatTile label="رد شده" value={formatNumber(summary.rejected)} />
            </div>
          )}

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Card>
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">هزینه به تفکیک نوع خدمت</h2>
              {byServiceType.length === 0 ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">درخواست پرداخت‌شده‌ای در این بازه نیست.</p>
              ) : (
                <div className="mt-4 h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={byServiceType} margin={{ top: 8, right: 8, left: 8, bottom: 8 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#94a3b8" strokeOpacity={0.35} vertical={false} />
                      <XAxis dataKey="service_type_name" tick={{ fontSize: 11 }} interval={0} angle={-15} textAnchor="end" height={50} />
                      <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => chartMoney(Number(v))} width={80} />
                      <Tooltip formatter={(v) => chartMoney(Number(v))} />
                      <Bar dataKey="total_paid" name="کل پرداختی" radius={[4, 4, 0, 0]} isAnimationActive={false}>
                        {byServiceType.map((_, i) => (
                          <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </Card>

            <Card>
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">هزینه به تفکیک ماه</h2>
              {byMonth.length === 0 ? (
                <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">درخواست پرداخت‌شده‌ای در این بازه نیست.</p>
              ) : (
                <div className="mt-4 h-72 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={byMonth} margin={{ top: 8, right: 8, left: 8, bottom: 8 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#94a3b8" strokeOpacity={0.35} vertical={false} />
                      <XAxis dataKey="month" tick={{ fontSize: 11 }} />
                      <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => chartMoney(Number(v))} width={80} />
                      <Tooltip formatter={(v) => chartMoney(Number(v))} />
                      <Line type="monotone" dataKey="total_paid" name="کل پرداختی" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} isAnimationActive={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              )}
            </Card>
          </div>

          <Card className="!p-0">
            <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-800">
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">هزینه به تفکیک کارمند</h2>
            </div>
            {byEmployee.length === 0 ? (
              <p className="p-5 text-sm text-slate-500 dark:text-slate-400">درخواست پرداخت‌شده‌ای در این بازه نیست.</p>
            ) : (
              <div className="scroll-x">
                <table className="w-full text-start text-sm">
                  <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                    <tr>
                      <th className="px-5 py-2 font-medium">کارمند</th>
                      <th className="px-5 py-2 font-medium">شماره پرسنلی</th>
                      <th className="px-5 py-2 font-medium">تعداد درخواست</th>
                      <th className="px-5 py-2 font-medium">کل پرداختی</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {byEmployee.map((row) => (
                      <tr key={row.employee_id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                        <td className="px-5 py-3 font-medium text-slate-900 dark:text-slate-100">{row.employee_name}</td>
                        <td className="px-5 py-3"><span className="dir-ltr">{row.personnel_no}</span></td>
                        <td className="px-5 py-3">{formatNumber(row.claim_count)}</td>
                        <td className="px-5 py-3">{formatMoney(row.total_paid)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
