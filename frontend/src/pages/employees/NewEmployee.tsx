import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { createEmployee } from '../../api/employees'
import { createUser } from '../../api/users'
import { apiErrorMessage } from '../../api/client'
import { listPlans } from '../../api/reference'
import type { CoveragePlan } from '../../api/types'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { PersianDateInput } from '../../components/PersianDateInput'
import { dateInputToRFC3339, todayYmd } from '../../lib/format'
import { inputClass, Field } from '../../components/FormField'

function suggestUsername(personnelNo: string) {
  return personnelNo.trim().toLowerCase().replace(/[^a-z0-9._-]/g, '')
}

export function NewEmployee() {
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [plans, setPlans] = useState<CoveragePlan[]>([])
  const [personnelNo, setPersonnelNo] = useState('')
  const [fullName, setFullName] = useState('')
  const [nationalId, setNationalId] = useState('')
  const [hireDate, setHireDate] = useState(todayYmd)
  const [department, setDepartment] = useState('')
  const [planId, setPlanId] = useState('')
  const [createLogin, setCreateLogin] = useState(true)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [usernameTouched, setUsernameTouched] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    listPlans().then(setPlans)
  }, [])

  function onPersonnelNoChange(value: string) {
    setPersonnelNo(value)
    if (!usernameTouched) setUsername(suggestUsername(value))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!personnelNo || !fullName || !nationalId || !hireDate || !department) {
      setError('همهٔ فیلدهای الزامی را تکمیل کنید.')
      return
    }
    if (createLogin && (!username.trim() || !password)) {
      setError('برای ساخت حساب ورود، نام کاربری و گذرواژه الزامی است.')
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
      if (createLogin) {
        try {
          await createUser({
            username: username.trim(),
            password,
            full_name: fullName,
            role: 'employee',
            employee_id: emp.id,
          })
          showToast('کارمند و حساب ورود ایجاد شد.', 'success')
        } catch (err) {
          showToast(
            `کارمند ساخته شد، ولی حساب ورود نه: ${apiErrorMessage(err)}. از پروندهٔ کارمند دوباره بسازید.`,
            'error',
          )
          navigate(`/employees/${emp.id}`)
          return
        }
      } else {
        showToast('کارمند ایجاد شد. در صورت نیاز بعداً از پرونده حساب بسازید.', 'success')
      }
      navigate(`/employees/${emp.id}`)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">کارمند جدید</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          پروندهٔ منابع انسانی. در صورت ساخت حساب ورود، کارمند می‌تواند خودش خسارت ثبت کند.
        </p>
      </div>
      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <ErrorBanner message={error} />}
          <Field label="شماره پرسنلی">
            <input value={personnelNo} onChange={(e) => onPersonnelNoChange(e.target.value)} className={inputClass} />
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

          <div className="rounded-lg border border-slate-200 p-4 dark:border-slate-700">
            <label className="flex cursor-pointer items-start gap-3">
              <input
                type="checkbox"
                checked={createLogin}
                onChange={(e) => setCreateLogin(e.target.checked)}
                className="mt-1 rounded border-slate-300 text-brand-600 focus:ring-brand-500"
              />
              <span>
                <span className="block text-sm font-medium text-slate-800 dark:text-slate-100">ایجاد حساب ورود</span>
                <span className="mt-0.5 block text-xs text-slate-500 dark:text-slate-400">
                  نام کاربری و گذرواژه برای ورود به سامانه با نقش کارمند.
                </span>
              </span>
            </label>
            {createLogin && (
              <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
                <Field label="نام کاربری">
                  <input
                    value={username}
                    onChange={(e) => {
                      setUsernameTouched(true)
                      setUsername(e.target.value)
                    }}
                    dir="ltr"
                    autoComplete="off"
                    className={inputClass}
                  />
                </Field>
                <Field label="گذرواژه">
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="new-password"
                    className={inputClass}
                  />
                </Field>
              </div>
            )}
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {submitting ? 'در حال ایجاد…' : createLogin ? 'ایجاد کارمند و حساب' : 'ایجاد کارمند'}
          </button>
        </form>
      </Card>
    </div>
  )
}
