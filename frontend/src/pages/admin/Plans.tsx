import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { createPlan, listContracts, listPlans } from '../../api/reference'
import { apiErrorMessage } from '../../api/client'
import type { CoveragePlan } from '../../api/types'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { DataTable } from '../../components/DataTable'
import type { Column } from '../../components/DataTable'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Field, inputClass } from '../../components/FormField'
import { qk } from '../../api/queryKeys'

export function Plans() {
  const { showToast } = useToast()
  const queryClient = useQueryClient()

  const plansQuery = useQuery({ queryKey: qk.plans(), queryFn: () => listPlans() })
  const contractsQuery = useQuery({ queryKey: qk.contracts(), queryFn: listContracts })
  const contracts = contractsQuery.data ?? []

  const [contractId, setContractId] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const create = useMutation({
    mutationFn: () => createPlan({ contract_id: contractId, name, description }),
    onSuccess: () => {
      showToast('طرح ایجاد شد.', 'success')
      setName('')
      setDescription('')
      // Invalidating the list replaces the old manual reload().
      void queryClient.invalidateQueries({ queryKey: qk.plans() })
    },
    onError: (err) => setFormError(apiErrorMessage(err)),
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    if (!contractId || !name) {
      setFormError('قرارداد و نام الزامی است.')
      return
    }
    create.mutate()
  }

  const columns: Column<CoveragePlan>[] = [
    { header: 'نام', cell: (p) => p.name },
    {
      header: 'قرارداد',
      cell: (p) => contracts.find((c) => c.id === p.contract_id)?.name ?? p.contract_id,
    },
    { header: 'توضیحات', cell: (p) => p.description },
  ]

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
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className={inputClass}
            />
          </Field>
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {create.isPending ? 'در حال ایجاد…' : 'ایجاد طرح'}
          </button>
        </form>
      </Card>

      <DataTable
        columns={columns}
        rows={plansQuery.data ?? []}
        rowKey={(p) => p.id}
        loading={plansQuery.isLoading}
        error={plansQuery.isError ? apiErrorMessage(plansQuery.error) : null}
        emptyMessage="هنوز طرحی ثبت نشده است."
      />
    </div>
  )
}
