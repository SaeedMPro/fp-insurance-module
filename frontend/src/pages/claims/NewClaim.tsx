import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { createClaim } from '../../api/claims'
import { apiErrorMessage } from '../../api/client'
import { listDependents, listEmployees } from '../../api/employees'
import type { BeneficiaryType, Dependent, Employee } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { useServiceTypes } from '../../hooks/useServiceTypes'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Field, inputClass } from '../../components/FormField'
import { PersianDateInput } from '../../components/PersianDateInput'
import { BENEFICIARY_LABELS, RELATION_LABELS, dateInputToRFC3339, todayYmd } from '../../lib/format'

export function NewClaim() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const { serviceTypes, loading: loadingServiceTypes } = useServiceTypes()
  const isAdmin = user?.role === 'admin'

  const [employees, setEmployees] = useState<Employee[]>([])
  const [employeeId, setEmployeeId] = useState(user?.employee_id ?? '')
  const [beneficiaryType, setBeneficiaryType] = useState<BeneficiaryType>('self')
  const [dependents, setDependents] = useState<Dependent[]>([])
  const [dependentId, setDependentId] = useState('')
  const [serviceTypeId, setServiceTypeId] = useState('')
  const [requestedAmount, setRequestedAmount] = useState('')
  const [receiptDate, setReceiptDate] = useState(todayYmd)
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (isAdmin) {
      listEmployees({ page_size: 200 }).then((res) => setEmployees(res.items))
    }
  }, [isAdmin])

  useEffect(() => {
    if (beneficiaryType !== 'dependent' || !employeeId) {
      setDependents([])
      setDependentId('')
      return
    }
    listDependents(employeeId).then(setDependents)
  }, [beneficiaryType, employeeId])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    if (!employeeId) {
      setError('کارمند را انتخاب کنید.')
      return
    }
    if (!serviceTypeId) {
      setError('نوع خدمت را انتخاب کنید.')
      return
    }
    if (beneficiaryType === 'dependent' && !dependentId) {
      setError('عضو تحت تکفل را انتخاب کنید.')
      return
    }
    const amount = Number(requestedAmount)
    if (!amount || amount <= 0) {
      setError('مبلغ درخواستی معتبر وارد کنید.')
      return
    }
    if (!receiptDate) {
      setError('تاریخ فاکتور را انتخاب کنید.')
      return
    }

    setSubmitting(true)
    try {
      const claim = await createClaim({
        employee_id: employeeId,
        beneficiary_type: beneficiaryType,
        dependent_id: beneficiaryType === 'dependent' ? dependentId : null,
        service_type_id: serviceTypeId,
        requested_amount: amount,
        receipt_date: dateInputToRFC3339(receiptDate),
        description,
      })
      navigate(`/claims/${claim.id}`)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">ثبت درخواست جدید</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          این یک پیش‌نویس ایجاد می‌کند؛ می‌توانید آن را از صفحهٔ جزئیات درخواست برای بررسی ثبت کنید.
        </p>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <ErrorBanner message={error} />}

          {isAdmin && (
            <Field label="کارمند">
              <select
                value={employeeId}
                onChange={(e) => setEmployeeId(e.target.value)}
                className={selectClass}
              >
                <option value="">انتخاب کارمند…</option>
                {employees.map((emp) => (
                  <option key={emp.id} value={emp.id}>
                    {emp.full_name} ({emp.personnel_no})
                  </option>
                ))}
              </select>
            </Field>
          )}

          <Field label="ذی‌نفع">
            <select
              value={beneficiaryType}
              onChange={(e) => setBeneficiaryType(e.target.value as BeneficiaryType)}
              className={selectClass}
            >
              <option value="self">{BENEFICIARY_LABELS.self}</option>
              <option value="dependent">{BENEFICIARY_LABELS.dependent}</option>
            </select>
          </Field>

          {beneficiaryType === 'dependent' && (
            <Field label="عضو تحت تکفل">
              <select
                value={dependentId}
                onChange={(e) => setDependentId(e.target.value)}
                className={selectClass}
              >
                <option value="">انتخاب عضو تحت تکفل…</option>
                {dependents.map((dep) => (
                  <option key={dep.id} value={dep.id}>
                    {dep.full_name} ({RELATION_LABELS[dep.relation]})
                  </option>
                ))}
              </select>
            </Field>
          )}

          <Field label="نوع خدمت">
            <select
              value={serviceTypeId}
              onChange={(e) => setServiceTypeId(e.target.value)}
              disabled={loadingServiceTypes}
              className={selectClass}
            >
              <option value="">انتخاب نوع خدمت…</option>
              {serviceTypes.map((st) => (
                <option key={st.id} value={st.id}>
                  {st.name}
                </option>
              ))}
            </select>
          </Field>

          <Field label="مبلغ درخواستی (ریال)">
            <input
              type="number"
              min="0"
              step="1"
              value={requestedAmount}
              onChange={(e) => setRequestedAmount(e.target.value)}
              className={inputClass}
            />
          </Field>

          <Field label="تاریخ فاکتور">
            <PersianDateInput value={receiptDate} onChange={setReceiptDate} />
          </Field>

          <Field label="توضیحات">
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className={inputClass}
            />
          </Field>

          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {submitting ? 'در حال ایجاد…' : 'ایجاد پیش‌نویس درخواست'}
          </button>
        </form>
      </Card>
    </div>
  )
}

const selectClass = inputClass
