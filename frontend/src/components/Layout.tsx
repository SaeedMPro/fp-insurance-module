import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'
import { useTheme } from '../hooks/useTheme'
import type { ThemePref } from '../hooks/useTheme'
import { navFor } from '../app/routes'
import { ROLE_LABELS } from '../lib/format'

// Nav items come from app/routes.ts — the single place that knows which role
// sees which screen (it also drives the router's guards).
const THEME_OPTIONS: { value: ThemePref; label: string }[] = [
  { value: 'light', label: 'روشن' },
  { value: 'dark', label: 'تیره' },
  { value: 'system', label: 'خودکار' },
]

export function Layout() {
  const { user, logout } = useAuth()
  const { pref, setPref } = useTheme()
  const navigate = useNavigate()

  if (!user) return null

  const items = navFor(user.role)

  function handleLogout() {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex min-h-screen bg-slate-50 dark:bg-slate-950">
      <aside className="flex w-60 shrink-0 flex-col border-e border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <div className="border-b border-slate-200 px-5 py-4 dark:border-slate-800">
          <div className="text-sm font-semibold text-slate-900 dark:text-slate-50">سامانه بیمه تکمیلی</div>
          <div className="text-xs text-slate-400 dark:text-slate-500">مطالبات و پوشش</div>
        </div>
        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {items.map((item) => (
            <NavLink
              key={item.label}
              to={item.to}
              end={item.to === '/claims'}
              className={({ isActive }) =>
                `block rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-brand-600 text-white'
                    : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t border-slate-200 px-5 py-4 dark:border-slate-800">
          <div className="text-sm font-medium text-slate-900 dark:text-slate-50">{user.full_name}</div>
          <div className="text-xs text-slate-400 dark:text-slate-500">{ROLE_LABELS[user.role]}</div>

          <div className="mt-3">
            <div className="text-xs text-slate-400 dark:text-slate-500">حالت نمایش</div>
            <div
              role="radiogroup"
              aria-label="حالت نمایش"
              className="mt-1 flex overflow-hidden rounded-lg border border-slate-200 text-xs dark:border-slate-700"
            >
              {THEME_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  role="radio"
                  aria-checked={pref === opt.value}
                  onClick={() => setPref(opt.value)}
                  className={`flex-1 px-2 py-1.5 font-medium transition-colors ${
                    pref === opt.value
                      ? 'bg-brand-600 text-white'
                      : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <button
            type="button"
            onClick={handleLogout}
            className="mt-3 w-full rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
          >
            خروج از حساب
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto px-8 py-8">
        <div className="mx-auto max-w-6xl">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
