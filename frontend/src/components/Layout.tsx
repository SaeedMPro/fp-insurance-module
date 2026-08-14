import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'
import { useTheme } from '../hooks/useTheme'
import { navFor } from '../app/routes'
import { ROLE_LABELS } from '../lib/format'

export function Layout() {
  const { user, logout } = useAuth()
  const { isDark, setPref } = useTheme()
  const navigate = useNavigate()

  if (!user) return null

  const items = navFor(user.role)

  function handleLogout() {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-950">
      <aside className="flex w-56 shrink-0 flex-col overflow-hidden border-e border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <div className="shrink-0 border-b border-slate-200 px-4 py-3 dark:border-slate-800">
          <div className="text-sm font-semibold text-slate-900 dark:text-slate-50">سامانه بیمه تکمیلی</div>
        </div>

        <nav className="min-h-0 flex-1 space-y-0.5 overflow-hidden px-2 py-3">
          {items.map((item) => (
            <NavLink
              key={item.label}
              to={item.to}
              end={item.to === '/claims'}
              className={({ isActive }) =>
                `block truncate rounded-md px-3 py-2 text-sm ${
                  isActive
                    ? 'bg-brand-600 font-medium text-white'
                    : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="shrink-0 space-y-1 border-t border-slate-200 px-3 py-3 dark:border-slate-800">
          <div className="truncate text-sm text-slate-800 dark:text-slate-100">{user.full_name}</div>
          <div className="truncate text-xs text-slate-400">{ROLE_LABELS[user.role]}</div>
          <div className="flex items-center justify-between pt-2">
            <button
              type="button"
              onClick={() => setPref(isDark ? 'light' : 'dark')}
              aria-pressed={isDark}
              aria-label={isDark ? 'تغییر به حالت روشن' : 'تغییر به حالت تیره'}
              className="rounded-md border border-slate-200 px-2.5 py-1 text-xs text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
            >
              {isDark ? 'روشن' : 'تیره'}
            </button>
            <button
              type="button"
              onClick={handleLogout}
              className="text-xs text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
            >
              خروج
            </button>
          </div>
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-y-auto px-8 py-8">
        <div className="mx-auto max-w-6xl">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
