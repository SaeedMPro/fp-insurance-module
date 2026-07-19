import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { createPlan, listContracts, listPlans } from '../../api/reference'
import { apiErrorMessage } from '../../api/client'
import type { CoveragePlan, InsuranceContract } from '../../api/types'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Field, inputClass } from '../../components/FormField'

export function Plans() {
  const { showToast } = useToast()
  const [plans, setPlans] = useState<CoveragePlan[]>([])
  const [contracts, setContracts] = useState<InsuranceContract[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [contractId, setContractId] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  function reload() {
    setLoading(true)
    listPlans()
      .then(setPlans)
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    reload()
    listContracts().then(setContracts)
  }, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    if (!contractId || !name) {
      setFormError('قرارداد و نام الزامی است.')
      return
    }
    setSubmitting(true)
    try {
      await createPlan({ contract_id: contractId, name, description })
      showToast('طرح ایجاد شد.', 'success')
      setName('')
      setDescription('')
      reload()
    } catch (err) {
      setFormError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">طرح‌های پوشش</h1>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">طرح جدید</h2>
        <form onSubmit={handleSubmit} className="mt-3 space-y-4">
          {formError && <ErrorBanner message={formError} />}
          <Field label="قرارداد">
            <select value={contractId} onChange={(e) => setContractId(e.target.value)} className={inputClass}>
              <option value="">انتخاب قرارداد…</option>
              {contracts.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="نام">
            <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
          </Field>
          <Field label="توضیحات">
            <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={3} className={inputClass} />
          </Field>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {submitting ? 'در حال ایجاد…' : 'ایجاد طرح'}
          </button>
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
        ) : plans.length === 0 ? (
          <p className="p-5 text-sm text-slate-500 dark:text-slate-400">هنوز طرحی ثبت نشده است.</p>
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">نام</th>
                  <th className="px-5 py-2 font-medium">قرارداد</th>
                  <th className="px-5 py-2 font-medium">توضیحات</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {plans.map((p) => (
                  <tr key={p.id}>
                    <td className="px-5 py-3">{p.name}</td>
                    <td className="px-5 py-3">{contracts.find((c) => c.id === p.contract_id)?.name ?? p.contract_id}</td>
                    <td className="px-5 py-3">{p.description}</td>
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
