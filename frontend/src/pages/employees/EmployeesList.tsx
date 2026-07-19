import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listEmployees } from '../../api/employees'
import { apiErrorMessage } from '../../api/client'
import type { Employee } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Pagination } from '../../components/Pagination'
import { formatDate, EMPLOYMENT_STATUS_LABELS } from '../../lib/format'

const PAGE_SIZE = 20

export function EmployeesList() {
  const { user } = useAuth()
  const [q, setQ] = useState('')
  const [employees, setEmployees] = useState<Employee[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    listEmployees({ q: q || undefined, page, page_size: PAGE_SIZE })
      .then((res) => {
        if (cancelled) return
        setEmployees(res.items)
        setTotal(res.total)
      })
      .catch((err) => !cancelled && setError(apiErrorMessage(err)))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
  }, [q, page])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">کارکنان</h1>
          <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">سوابق پرسنلی و تخصیص طرح.</p>
        </div>
        {user?.role === 'admin' && (
          <Link
            to="/employees/new"
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            کارمند جدید
          </Link>
        )}
      </div>

      <Card className="!p-0">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          <input
            type="search"
            placeholder="جستجو بر اساس نام یا شماره پرسنلی…"
            value={q}
            onChange={(e) => {
              setQ(e.target.value)
              setPage(1)
            }}
            className="w-full max-w-sm rounded-lg border border-slate-300 px-3 py-1.5 text-sm text-slate-900 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
          />
        </div>

        {error && (
          <div className="p-5">
            <ErrorBanner message={error} />
          </div>
        )}
        {loading ? (
          <Spinner />
        ) : employees.length === 0 ? (
          <p className="p-5 text-sm text-slate-500 dark:text-slate-400">کارمندی یافت نشد.</p>
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">شماره پرسنلی</th>
                  <th className="px-5 py-2 font-medium">نام و نام خانوادگی</th>
                  <th className="px-5 py-2 font-medium">واحد سازمانی</th>
                  <th className="px-5 py-2 font-medium">تاریخ استخدام</th>
                  <th className="px-5 py-2 font-medium">وضعیت</th>
                  <th className="px-5 py-2 font-medium" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {employees.map((emp) => (
                  <tr key={emp.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="px-5 py-3"><span className="dir-ltr">{emp.personnel_no}</span></td>
                    <td className="px-5 py-3">{emp.full_name}</td>
                    <td className="px-5 py-3">{emp.department}</td>
                    <td className="px-5 py-3">{formatDate(emp.hire_date)}</td>
                    <td className="px-5 py-3">{EMPLOYMENT_STATUS_LABELS[emp.employment_status]}</td>
                    <td className="px-5 py-3 text-end">
                      <Link to={`/employees/${emp.id}`} className="font-medium text-brand-600 hover:underline">
                        مشاهده
                      </Link>
                    </td>
                  </tr>
                ))}
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
