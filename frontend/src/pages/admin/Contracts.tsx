import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { createContract, listContracts } from '../../api/reference'
import { apiErrorMessage } from '../../api/client'
import type { InsuranceContract } from '../../api/types'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Field, inputClass } from '../../components/FormField'
import { dateInputToRFC3339, formatDate } from '../../lib/format'

export function Contracts() {
  const { showToast } = useToast()
  const [contracts, setContracts] = useState<InsuranceContract[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  function reload() {
    setLoading(true)
    listContracts()
      .then(setContracts)
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(reload, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    if (!name || !startDate || !endDate) {
      setFormError('نام، تاریخ شروع و تاریخ پایان الزامی است.')
      return
    }
    setSubmitting(true)
    try {
      await createContract({
        name,
        start_date: dateInputToRFC3339(startDate),
        end_date: dateInputToRFC3339(endDate),
        is_active: isActive,
      })
      showToast('قرارداد ایجاد شد.', 'success')
      setName('')
      setStartDate('')
      setEndDate('')
      setIsActive(true)
      reload()
    } catch (err) {
      setFormError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">قراردادها</h1>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">قرارداد جدید</h2>
        <form onSubmit={handleSubmit} className="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2">
          {formError && (
            <div className="sm:col-span-2">
              <ErrorBanner message={formError} />
            </div>
          )}
          <Field label="نام">
            <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
          </Field>
          <Field label="فعال">
            <select value={isActive ? 'yes' : 'no'} onChange={(e) => setIsActive(e.target.value === 'yes')} className={inputClass}>
              <option value="yes">بله</option>
              <option value="no">خیر</option>
            </select>
          </Field>
          <Field label="تاریخ شروع">
            <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className={inputClass} />
          </Field>
          <Field label="تاریخ پایان">
            <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className={inputClass} />
          </Field>
          <div className="sm:col-span-2">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting ? 'در حال ایجاد…' : 'ایجاد قرارداد'}
            </button>
          </div>
        </form>
      </Card>

      <Card className="!p-0">
        {error && (
          <div className="p-5">
            <ErrorBanner message={error} />
          </div>
        )}
        {loading ? (
          <Spinner />
        ) : contracts.length === 0 ? (
          <p className="p-5 text-sm text-slate-500 dark:text-slate-400">هنوز قراردادی ثبت نشده است.</p>
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">نام</th>
                  <th className="px-5 py-2 font-medium">تاریخ شروع</th>
                  <th className="px-5 py-2 font-medium">تاریخ پایان</th>
                  <th className="px-5 py-2 font-medium">فعال</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {contracts.map((c) => (
                  <tr key={c.id}>
                    <td className="px-5 py-3">{c.name}</td>
                    <td className="px-5 py-3">{formatDate(c.start_date)}</td>
                    <td className="px-5 py-3">{formatDate(c.end_date)}</td>
                    <td className="px-5 py-3">{c.is_active ? 'بله' : 'خیر'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
