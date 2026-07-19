import { useEffect, useState } from 'react'
import { getRemainingCaps } from '../../api/employees'
import { apiErrorMessage } from '../../api/client'
import type { RemainingCap } from '../../api/types'
import { useAuth } from '../../context/useAuth'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { formatMoney } from '../../lib/format'

export function MyCoverage() {
  const { user } = useAuth()
  const [caps, setCaps] = useState<RemainingCap[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user?.employee_id) {
      setLoading(false)
      setError('حساب شما به هیچ پروندهٔ کارمندی متصل نیست.')
      return
    }
    getRemainingCaps(user.employee_id)
      .then(setCaps)
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }, [user])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">پوشش بیمه‌ای من</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          سقف‌های باقی‌ماندهٔ سالانه به تفکیک نوع خدمت در طرح فعلی شما.
        </p>
      </div>

      {error && <ErrorBanner message={error} />}
      {loading ? (
        <Spinner />
      ) : caps.length === 0 ? (
        <p className="text-sm text-slate-500 dark:text-slate-400">هنوز قانون پوششی برای طرح شما تعریف نشده است.</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {caps.map((cap) => {
            const usedPct =
              cap.annual_cap && cap.annual_cap > 0
                ? Math.min(100, Math.round((cap.used_annual / cap.annual_cap) * 100))
                : 0
            return (
              <Card key={cap.service_type_code}>
                <div className="flex items-center justify-between">
                  <h2 className="font-semibold text-slate-900 dark:text-slate-50">{cap.service_type_name}</h2>
                  <span className="text-sm font-medium text-brand-600">{cap.coverage_percent}٪</span>
                </div>
                <dl className="mt-3 space-y-1 text-sm text-slate-500 dark:text-slate-400">
                  <div className="flex justify-between">
                    <dt>سقف هر دفعه</dt>
                    <dd className="text-slate-800 dark:text-slate-200">{formatMoney(cap.per_claim_cap)}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>سقف سالانه</dt>
                    <dd className="text-slate-800 dark:text-slate-200">{formatMoney(cap.annual_cap)}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>مصرف‌شده امسال</dt>
                    <dd className="text-slate-800 dark:text-slate-200">{formatMoney(cap.used_annual)}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>باقی‌مانده</dt>
                    <dd className="font-medium text-emerald-600 dark:text-emerald-400">
                      {formatMoney(cap.remaining_annual)}
                    </dd>
                  </div>
                </dl>
                {cap.annual_cap != null && cap.annual_cap > 0 && (
                  <div className="mt-3">
                    <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
                      <div className="h-full rounded-full bg-brand-500" style={{ width: `${usedPct}%` }} />
                    </div>
                    <div className="mt-1 text-xs text-slate-400 dark:text-slate-500">{usedPct}٪ مصرف‌شده</div>
                  </div>
                )}
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
