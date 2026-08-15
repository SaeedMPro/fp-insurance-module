import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { createServiceType, listServiceTypes } from '../../api/reference'
import { apiErrorMessage } from '../../api/client'
import type { ServiceType } from '../../api/types'
import { primeServiceTypesCache } from '../../hooks/useServiceTypes'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { DataTable } from '../../components/DataTable'
import type { Column } from '../../components/DataTable'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Field, inputClass } from '../../components/FormField'
import { ROUTES } from '../../app/routes'
import { formatDate } from '../../lib/format'

export function ServiceTypes() {
  const { showToast } = useToast()
  const [items, setItems] = useState<ServiceType[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  function reload() {
    setLoading(true)
    listServiceTypes()
      .then((data) => {
        primeServiceTypesCache(data)
        setItems(data)
      })
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(reload, [])

  function resetForm() {
    setCode('')
    setName('')
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    const trimmedCode = code.trim()
    const trimmedName = name.trim()
    if (!trimmedCode || !trimmedName) {
      setFormError('نام و کد سیستمی الزامی است.')
      return
    }
    setSubmitting(true)
    try {
      await createServiceType({ code: trimmedCode, name: trimmedName })
      showToast('نوع خدمت ایجاد شد.', 'success')
      resetForm()
      reload()
    } catch (err) {
      setFormError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  const columns: Column<ServiceType>[] = [
    { header: 'نام', cell: (st) => st.name },
    {
      header: 'کد',
      cell: (st) => (
        <code dir="ltr" className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-300">
          {st.code}
        </code>
      ),
    },
    {
      header: 'ایجاد',
      cell: (st) => <span className="text-slate-500 dark:text-slate-400">{formatDate(st.created_at)}</span>,
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">انواع خدمت</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          دسته‌های قابل‌خسارت سامانه. پس از افزودن خدمت، برای هر طرح یک{' '}
          <Link to={ROUTES.coverageRules.path} className="text-brand-600 hover:underline dark:text-brand-400">
            قانون پوشش
          </Link>{' '}
          تعریف کنید تا قیمت‌گذاری فعال شود.
        </p>
      </div>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">خدمت جدید</h2>
        <form onSubmit={handleSubmit} className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          {formError && (
            <div className="sm:col-span-2">
              <ErrorBanner message={formError} />
            </div>
          )}

          <Field label="نام">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="فیزیوتراپی"
              autoComplete="off"
              className={inputClass}
            />
          </Field>

          <Field label="کد سیستمی">
            <input
              value={code}
              onChange={(e) => setCode(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '').slice(0, 30))}
              placeholder="physiotherapy"
              dir="ltr"
              autoComplete="off"
              spellCheck={false}
              className={`${inputClass} font-mono`}
            />
            <span className="mt-1 block text-xs font-normal text-slate-500 dark:text-slate-400">
              شناسهٔ ثابت برای سیستم (حروف کوچک، عدد، زیرخط).
            </span>
          </Field>

          <div className="sm:col-span-2">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting ? 'در حال ایجاد…' : 'ایجاد خدمت'}
            </button>
          </div>
        </form>
      </Card>

      <DataTable
        title="فهرست انواع خدمت"
        columns={columns}
        rows={items}
        rowKey={(st) => st.id}
        loading={loading}
        error={error}
        emptyMessage="هنوز نوع خدمتی ثبت نشده است."
      />
    </div>
  )
}
