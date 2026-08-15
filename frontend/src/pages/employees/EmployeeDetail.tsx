import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import { createDependent, getEmployee, listDependents, updateEmployee } from '../../api/employees'
import { apiErrorMessage } from '../../api/client'
import { listPlans } from '../../api/reference'
import type { CoveragePlan, Dependent, Employee, Relation } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Field, inputClass } from '../../components/FormField'
import { PersianDateInput } from '../../components/PersianDateInput'
import { formatDate, RELATION_LABELS } from '../../lib/format'

export function EmployeeDetail() {
  const { id } = useParams<{ id: string }>()
  const { user } = useAuth()
  const { showToast } = useToast()
  const isAdmin = user?.role === 'admin'

  const [employee, setEmployee] = useState<Employee | null>(null)
  const [plans, setPlans] = useState<CoveragePlan[]>([])
  const [dependents, setDependents] = useState<Dependent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [fullName, setFullName] = useState('')
  const [department, setDepartment] = useState('')
  const [planId, setPlanId] = useState('')
  const [employmentStatus, setEmploymentStatus] = useState('active')
  const [savingProfile, setSavingProfile] = useState(false)

  const [depName, setDepName] = useState('')
  const [depRelation, setDepRelation] = useState<Relation>('spouse')
  const [depNationalId, setDepNationalId] = useState('')
  const [depBirthDate, setDepBirthDate] = useState('')
  const [savingDependent, setSavingDependent] = useState(false)

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const [emp, deps] = await Promise.all([getEmployee(id), listDependents(id)])
      setEmployee(emp)
      setDependents(deps)
      setFullName(emp.full_name)
      setDepartment(emp.department)
      setPlanId(emp.plan_id ?? '')
      setEmploymentStatus(emp.employment_status)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    load()
    listPlans().then(setPlans)
  }, [load])

  async function handleSaveProfile(e: FormEvent) {
    e.preventDefault()
    if (!id) return
    setSavingProfile(true)
    try {
      const updated = await updateEmployee(id, {
        full_name: fullName,
        department,
        plan_id: planId || null,
        employment_status: employmentStatus,
      })
      setEmployee(updated)
      showToast('کارمند به‌روزرسانی شد.', 'success')
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    } finally {
      setSavingProfile(false)
    }
  }

  async function handleAddDependent(e: FormEvent) {
    e.preventDefault()
    if (!id) return
    if (!depName || !depNationalId || !depBirthDate) {
      showToast('همهٔ فیلدهای عضو تحت تکفل را تکمیل کنید.', 'error')
      return
    }
    setSavingDependent(true)
    try {
      const dep = await createDependent(id, {
        full_name: depName,
        relation: depRelation as 'spouse' | 'child' | 'parent',
        national_id: depNationalId,
        birth_date: `${depBirthDate}T00:00:00Z`,
      })
      setDependents((prev) => [...prev, dep])
      setDepName('')
      setDepNationalId('')
      setDepBirthDate('')
      showToast('عضو تحت تکفل افزوده شد.', 'success')
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    } finally {
      setSavingDependent(false)
    }
  }

  if (loading) return <Spinner />
  if (error && !employee) return <ErrorBanner message={error} />
  if (!employee) return null

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">{employee.full_name}</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          شماره پرسنلی <span className="dir-ltr">{employee.personnel_no}</span> · کد ملی <span className="dir-ltr">{employee.national_id}</span> · تاریخ استخدام {formatDate(employee.hire_date)}
        </p>
      </div>

      {error && <ErrorBanner message={error} />}

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">مشخصات</h2>
        <form onSubmit={handleSaveProfile} className="mt-3 space-y-4">
          <Field label="نام و نام خانوادگی">
            <input
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              disabled={!isAdmin}
              className={inputClass}
            />
          </Field>
          <Field label="واحد سازمانی">
            <input
              value={department}
              onChange={(e) => setDepartment(e.target.value)}
              disabled={!isAdmin}
              className={inputClass}
            />
          </Field>
          <Field label="طرح">
            <select value={planId} onChange={(e) => setPlanId(e.target.value)} disabled={!isAdmin} className={inputClass}>
              <option value="">بدون طرح</option>
              {plans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="وضعیت اشتغال">
            <select
              value={employmentStatus}
              onChange={(e) => setEmploymentStatus(e.target.value)}
              disabled={!isAdmin}
              className={inputClass}
            >
              <option value="active">شاغل</option>
              <option value="terminated">پایان همکاری</option>
            </select>
          </Field>
          {isAdmin && (
            <button
              type="submit"
              disabled={savingProfile}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {savingProfile ? 'در حال ذخیره…' : 'ذخیره تغییرات'}
            </button>
          )}
        </form>
      </Card>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">اعضای تحت تکفل</h2>
        {dependents.length === 0 ? (
          <p className="mt-2 text-sm text-slate-400 dark:text-slate-500">عضو تحت تکفلی ثبت نشده است.</p>
        ) : (
          <ul className="mt-3 divide-y divide-slate-100 text-sm dark:divide-slate-800">
            {dependents.map((dep) => (
              <li key={dep.id} className="flex items-center justify-between py-2">
                <span className="text-slate-800 dark:text-slate-100">{dep.full_name}</span>
                <span className="text-slate-400 dark:text-slate-500">{RELATION_LABELS[dep.relation]}</span>
              </li>
            ))}
          </ul>
        )}

        {isAdmin && (
          <form onSubmit={handleAddDependent} className="mt-4 grid grid-cols-2 gap-3 border-t border-slate-100 pt-4 dark:border-slate-800">
            <Field label="نام و نام خانوادگی">
              <input value={depName} onChange={(e) => setDepName(e.target.value)} className={inputClass} />
            </Field>
            <Field label="نسبت">
              <select value={depRelation} onChange={(e) => setDepRelation(e.target.value as Relation)} className={inputClass}>
                <option value="spouse">همسر</option>
                <option value="child">فرزند</option>
                <option value="parent">والدین</option>
              </select>
            </Field>
            <Field label="کد ملی">
              <input value={depNationalId} onChange={(e) => setDepNationalId(e.target.value)} className={inputClass} />
            </Field>
            <Field label="تاریخ تولد">
              <PersianDateInput value={depBirthDate} onChange={setDepBirthDate} />
            </Field>
            <div className="col-span-2">
              <button
                type="submit"
                disabled={savingDependent}
                className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
              >
                {savingDependent ? 'در حال افزودن…' : 'افزودن عضو تحت تکفل'}
              </button>
            </div>
          </form>
        )}
      </Card>
    </div>
  )
}
