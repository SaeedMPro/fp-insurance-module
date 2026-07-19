import { Link } from 'react-router-dom'

export function NotFound() {
  return (
    <div className="py-16 text-center">
      <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-50">صفحه یافت نشد</h1>
      <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
        صفحه‌ای که به دنبال آن هستید وجود ندارد.
      </p>
      <Link to="/" className="mt-4 inline-block text-sm font-medium text-brand-600 hover:underline">
        بازگشت به صفحهٔ اصلی
      </Link>
    </div>
  )
}
