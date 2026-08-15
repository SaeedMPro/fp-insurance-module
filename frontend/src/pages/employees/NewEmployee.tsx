import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { createEmployee } from '../../api/employees'
import { apiErrorMessage } from '../../api/client'
import { listPlans } from '../../api/reference'
import type { CoveragePlan } from '../../api/types'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { PersianDateInput } from '../../components/PersianDateInput'
import { dateInputToRFC3339, todayYmd } from '../../lib/format'
import { inputClass, Field } from '../../components/FormField'

export function NewEmployee() {
  const navigate = useNavigate()
  const [plans, setPlans] = useState<CoveragePlan[]>([])
  const [personnelNo, setPersonnelNo] = useState('')
  const [fullName, setFullName] = useState('')
  const [nationalId, setNationalId] = useState('')
  const [hireDate, setHireDate] = useState(todayYmd)
  const [department, setDepartment] = useState('')
  const [planId, setPlanId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    listPlans().then(setPlans)
  }, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!personnelNo || !fullName || !nationalId || !hireDate || !department) {
      setError('همهٔ فیلدهای الزامی را تکمیل کنید.')
      return
    }
    setSubmitting(true)
    try {
      const emp = await createEmployee({
        personnel_no: personnelNo,
        full_name: fullName,
        national_id: nationalId,
        hire_date: dateInputToRFC3339(hireDate),
        department,
        plan_id: planId || null,
      })
      navigate(`/employees/${emp.id}`)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-xl space-y-6">
      <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">کارمند جدید</h1>
      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <ErrorBanner message={error} />}
          <Field label="شماره پرسنلی">
            <input value={personnelNo} onChange={(e) => setPersonnelNo(e.target.value)} className={inputClass} />
          </Field>
          <Field label="نام و نام خانوادگی">
            <input value={fullName} onChange={(e) => setFullName(e.target.value)} className={inputClass} />
          </Field>
          <Field label="کد ملی">
            <input value={nationalId} onChange={(e) => setNationalId(e.target.value)} className={inputClass} />
          </Field>
          <Field label="تاریخ استخدام">
            <PersianDateInput value={hireDate} onChange={setHireDate} />
          </Field>
          <Field label="واحد سازمانی">
            <input value={department} onChange={(e) => setDepartment(e.target.value)} className={inputClass} />
          </Field>
          <Field label="طرح">
            <select value={planId} onChange={(e) => setPlanId(e.target.value)} className={inputClass}>
              <option value="">بدون طرح</option>
              {plans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name}
                </option>
              ))}
            </select>
          </Field>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {submitting ? 'در حال ایجاد…' : 'ایجاد کارمند'}
          </button>
        </form>
      </Card>
    </div>
  )
}
