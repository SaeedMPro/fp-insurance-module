import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { createCoverageRule, listCoverageRules, listPlans } from '../../api/reference'
import { apiErrorMessage } from '../../api/client'
import type { CoveragePlan, CoverageRule, Relation } from '../../api/types'
import { useServiceTypes } from '../../hooks/useServiceTypes'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Field, inputClass } from '../../components/FormField'
import { dateInputToRFC3339, formatDate, formatMoney, formatNumber, RELATION_LABELS } from '../../lib/format'

const RELATIONS: Relation[] = ['self', 'spouse', 'child', 'parent']

export function CoverageRules() {
  const { serviceTypes, byId: serviceTypeById } = useServiceTypes()
  const { showToast } = useToast()

  const [plans, setPlans] = useState<CoveragePlan[]>([])
  const [rules, setRules] = useState<CoverageRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [planId, setPlanId] = useState('')
  const [serviceTypeId, setServiceTypeId] = useState('')
  const [coveragePercent, setCoveragePercent] = useState('')
  const [perClaimCap, setPerClaimCap] = useState('')
  const [annualCap, setAnnualCap] = useState('')
  const [waitingPeriodDays, setWaitingPeriodDays] = useState('0')
  const [eligibleRelations, setEligibleRelations] = useState<Relation[]>(['self'])
  const [effectiveFrom, setEffectiveFrom] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  function reload() {
    setLoading(true)
    setError(null)
    listCoverageRules()
      .then(setRules)
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    reload()
    listPlans().then(setPlans)
  }, [])

  function toggleRelation(rel: Relation) {
    setEligibleRelations((prev) => (prev.includes(rel) ? prev.filter((r) => r !== rel) : [...prev, rel]))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    if (!planId || !serviceTypeId || !coveragePercent || !effectiveFrom) {
      setFormError('طرح، نوع خدمت، درصد پوشش و تاریخ اعمال الزامی است.')
      return
    }
    if (eligibleRelations.length === 0) {
      setFormError('حداقل یک نسبت مجاز را انتخاب کنید.')
      return
    }
    setSubmitting(true)
    try {
      await createCoverageRule({
        plan_id: planId,
        service_type_id: serviceTypeId,
        coverage_percent: Number(coveragePercent),
        per_claim_cap: perClaimCap ? Number(perClaimCap) : null,
        annual_cap: annualCap ? Number(annualCap) : null,
        waiting_period_days: Number(waitingPeriodDays) || 0,
        eligible_relations: eligibleRelations,
        effective_from: dateInputToRFC3339(effectiveFrom),
      })
      showToast('نسخهٔ جدید قانون پوشش منتشر شد — تغییرات بلافاصله اعمال می‌شوند.', 'success')
      setCoveragePercent('')
      setPerClaimCap('')
      setAnnualCap('')
      setWaitingPeriodDays('0')
      setEffectiveFrom('')
      reload()
    } catch (err) {
      setFormError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">قوانین پوشش</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          قوانین پوشش، موتور سیاست‌گذاری داده‌محور هستند: انتشار نسخهٔ جدید در پایین، بلافاصله و بدون تغییر کد تعیین
          می‌کند رول انجین چه درخواست‌هایی را تأیید و چه مبلغی را پرداخت کند. نسخهٔ قبلی برای همان طرح و نوع خدمت
          به‌طور خودکار بسته می‌شود.
        </p>
      </div>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">انتشار نسخهٔ جدید قانون</h2>
        <form onSubmit={handleSubmit} className="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2">
          {formError && (
            <div className="sm:col-span-2">
              <ErrorBanner message={formError} />
            </div>
          )}
          <Field label="طرح">
            <select value={planId} onChange={(e) => setPlanId(e.target.value)} className={inputClass}>
              <option value="">انتخاب طرح…</option>
              {plans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="نوع خدمت">
            <select value={serviceTypeId} onChange={(e) => setServiceTypeId(e.target.value)} className={inputClass}>
              <option value="">انتخاب نوع خدمت…</option>
              {serviceTypes.map((st) => (
                <option key={st.id} value={st.id}>
                  {st.name_fa || st.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="درصد پوشش">
            <input
              type="number"
              min="0"
              max="100"
              value={coveragePercent}
              onChange={(e) => setCoveragePercent(e.target.value)}
              className={inputClass}
            />
          </Field>
          <Field label="دورهٔ انتظار (روز)">
            <input
              type="number"
              min="0"
              value={waitingPeriodDays}
              onChange={(e) => setWaitingPeriodDays(e.target.value)}
              className={inputClass}
            />
          </Field>
          <Field label="سقف هر دفعه (ریال، اختیاری)">
            <input type="number" min="0" value={perClaimCap} onChange={(e) => setPerClaimCap(e.target.value)} className={inputClass} />
          </Field>
          <Field label="سقف سالانه (ریال، اختیاری)">
            <input type="number" min="0" value={annualCap} onChange={(e) => setAnnualCap(e.target.value)} className={inputClass} />
          </Field>
          <Field label="تاریخ اعمال از">
            <input type="date" value={effectiveFrom} onChange={(e) => setEffectiveFrom(e.target.value)} className={inputClass} />
          </Field>
          <div>
            <span className="block text-sm font-medium text-slate-700 dark:text-slate-300">نسبت‌های مجاز</span>
            <div className="mt-1 flex flex-wrap gap-3">
              {RELATIONS.map((rel) => (
                <label key={rel} className="flex items-center gap-1.5 text-sm text-slate-600 dark:text-slate-300">
                  <input
                    type="checkbox"
                    checked={eligibleRelations.includes(rel)}
                    onChange={() => toggleRelation(rel)}
                    className="rounded border-slate-300"
                  />
                  <span>{RELATION_LABELS[rel]}</span>
                </label>
              ))}
            </div>
          </div>
          <div className="sm:col-span-2">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting ? 'در حال انتشار…' : 'انتشار نسخهٔ قانون'}
            </button>
          </div>
        </form>
      </Card>

      <Card className="!p-0">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">تاریخچهٔ نسخه‌های قانون</h2>
        </div>
        {error && (
          <div className="p-5">
            <ErrorBanner message={error} />
          </div>
        )}
        {loading ? (
          <Spinner />
        ) : rules.length === 0 ? (
          <p className="p-5 text-sm text-slate-500 dark:text-slate-400">هنوز قانون پوششی ثبت نشده است.</p>
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">طرح</th>
                  <th className="px-5 py-2 font-medium">نوع خدمت</th>
                  <th className="px-5 py-2 font-medium">درصد پوشش</th>
                  <th className="px-5 py-2 font-medium">سقف هر دفعه</th>
                  <th className="px-5 py-2 font-medium">سقف سالانه</th>
                  <th className="px-5 py-2 font-medium">روزهای انتظار</th>
                  <th className="px-5 py-2 font-medium">نسبت‌ها</th>
                  <th className="px-5 py-2 font-medium">تاریخ اعمال از</th>
                  <th className="px-5 py-2 font-medium">تاریخ اعمال تا</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {rules.map((rule) => {
                  const plan = plans.find((p) => p.id === rule.plan_id)
                  const st = serviceTypeById.get(rule.service_type_id)
                  const isActive = !rule.effective_to
                  return (
                    <tr key={rule.id} className={isActive ? 'bg-emerald-50/50 dark:bg-emerald-950/20' : ''}>
                      <td className="px-5 py-3">{plan?.name ?? rule.plan_id}</td>
                      <td className="px-5 py-3">{st ? st.name_fa || st.name : rule.service_type_id}</td>
                      <td className="px-5 py-3">{formatNumber(rule.coverage_percent)}٪</td>
                      <td className="px-5 py-3">{formatMoney(rule.per_claim_cap)}</td>
                      <td className="px-5 py-3">{formatMoney(rule.annual_cap)}</td>
                      <td className="px-5 py-3">{formatNumber(rule.waiting_period_days)}</td>
                      <td className="px-5 py-3">{rule.eligible_relations.map((r) => RELATION_LABELS[r]).join('، ')}</td>
                      <td className="px-5 py-3">{formatDate(rule.effective_from)}</td>
                      <td className="px-5 py-3">{rule.effective_to ? formatDate(rule.effective_to) : 'فعال'}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
