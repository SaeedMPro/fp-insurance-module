import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { createUser, listUsers, updateUser } from '../../api/users'
import { listEmployees } from '../../api/employees'
import { apiErrorMessage } from '../../api/client'
import type { Employee, Role, User } from '../../api/types'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { Spinner } from '../../components/Spinner'
import { Field, inputClass } from '../../components/FormField'
import { ROLE_LABELS } from '../../lib/format'

const ASSIGNABLE_ROLES: Role[] = ['reviewer', 'employee', 'auditor']
const MIN_PASSWORD_LEN = 8

export function Users() {
  const { showToast } = useToast()

  const [users, setUsers] = useState<User[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create form
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [role, setRole] = useState<Role>('employee')
  const [employeeId, setEmployeeId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  // Password reset dialog
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [resetPassword, setResetPassword] = useState('')
  const [resetConfirm, setResetConfirm] = useState('')
  const [resetError, setResetError] = useState<string | null>(null)
  const [resetSubmitting, setResetSubmitting] = useState(false)

  function reload() {
    setLoading(true)
    setError(null)
    listUsers()
      .then(setUsers)
      .catch((err) => setError(apiErrorMessage(err)))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    reload()
    listEmployees({ page_size: 200 })
      .then((res) => setEmployees(res.items))
      .catch(() => setEmployees([]))
  }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    if (!username || !password || !fullName) {
      setFormError('نام کاربری، گذرواژه و نام و نام خانوادگی الزامی است.')
      return
    }
    if (password.length < MIN_PASSWORD_LEN) {
      setFormError(`گذرواژه باید حداقل ${MIN_PASSWORD_LEN} کاراکتر باشد.`)
      return
    }
    if (role === 'employee' && !employeeId) {
      setFormError('برای نقش کارمند، انتخاب پروندهٔ کارمندی الزامی است.')
      return
    }
    setSubmitting(true)
    try {
      await createUser({
        username,
        password,
        full_name: fullName,
        role,
        employee_id: role === 'employee' ? employeeId : null,
      })
      showToast('کاربر ایجاد شد.', 'success')
      setUsername('')
      setPassword('')
      setFullName('')
      setRole('employee')
      setEmployeeId('')
      reload()
    } catch (err) {
      setFormError(apiErrorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRoleChange(user: User, newRole: Role) {
    if (newRole === 'employee' && !user.employee_id) {
      showToast('این حساب به پروندهٔ کارمندی وصل نیست؛ کاربر کارمند را از فرم ایجاد با پیوند بسازید.', 'error')
      reload()
      return
    }
    if (!window.confirm(`نقش «${user.username}» به «${ROLE_LABELS[newRole]}» تغییر کند؟`)) {
      reload()
      return
    }
    try {
      await updateUser(user.id, { role: newRole })
      showToast(`نقش ${user.username} به‌روزرسانی شد.`, 'success')
      reload()
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
      reload()
    }
  }

  async function handleToggleActive(user: User) {
    if (user.role === 'admin') {
      showToast('حساب مدیر سامانه را نمی‌توان غیرفعال کرد.', 'error')
      return
    }
    try {
      await updateUser(user.id, { is_active: !user.is_active })
      showToast(`کاربر ${user.username} ${user.is_active ? 'غیرفعال شد' : 'فعال شد'}.`, 'success')
      reload()
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    }
  }

  function openResetPassword(user: User) {
    setResetTarget(user)
    setResetPassword('')
    setResetConfirm('')
    setResetError(null)
  }

  function closeResetPassword() {
    if (resetSubmitting) return
    setResetTarget(null)
    setResetPassword('')
    setResetConfirm('')
    setResetError(null)
  }

  async function handleResetPassword(e: FormEvent) {
    e.preventDefault()
    if (!resetTarget) return
    setResetError(null)
    if (resetPassword.length < MIN_PASSWORD_LEN) {
      setResetError(`گذرواژه باید حداقل ${MIN_PASSWORD_LEN} کاراکتر باشد.`)
      return
    }
    if (resetPassword !== resetConfirm) {
      setResetError('گذرواژه و تکرار آن یکسان نیستند.')
      return
    }
    setResetSubmitting(true)
    try {
      await updateUser(resetTarget.id, { password: resetPassword })
      showToast(`گذرواژهٔ ${resetTarget.username} بازنشانی شد.`, 'success')
      setResetTarget(null)
      setResetPassword('')
      setResetConfirm('')
    } catch (err) {
      setResetError(apiErrorMessage(err))
    } finally {
      setResetSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">کاربران</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          حساب ورود به سامانه. نقش «کارمند» حتماً باید به یک ردیف در فهرست کارکنان وصل شود تا بتواند
          خسارت ثبت کند. کارکنان بدون حساب (مثلاً از همگام‌سازی HR) فقط توسط ادمین مدیریت می‌شوند.
          مدیر سامانه فقط از طریق seed یا <span className="dir-ltr font-mono text-xs">make create-admin</span>{' '}
          ساخته می‌شود و نمی‌توان او را غیرفعال یا تغییر نقش داد.
        </p>
      </div>

      <Card>
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">ایجاد کاربر</h2>
        <form onSubmit={handleCreate} className="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2">
          {formError && (
            <div className="sm:col-span-2">
              <ErrorBanner message={formError} />
            </div>
          )}
          <Field label="نام کاربری">
            <input value={username} onChange={(e) => setUsername(e.target.value)} className={inputClass} autoComplete="off" />
          </Field>
          <Field label="نام و نام خانوادگی">
            <input value={fullName} onChange={(e) => setFullName(e.target.value)} className={inputClass} />
          </Field>
          <Field label="گذرواژه">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputClass}
              autoComplete="new-password"
              minLength={MIN_PASSWORD_LEN}
            />
          </Field>
          <Field label="نقش">
            <select value={role} onChange={(e) => setRole(e.target.value as Role)} className={inputClass}>
              {ASSIGNABLE_ROLES.map((r) => (
                <option key={r} value={r}>
                  {ROLE_LABELS[r]}
                </option>
              ))}
            </select>
          </Field>
          {role === 'employee' && (
            <Field label="کارمند مرتبط">
              <select value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} className={inputClass} required>
                <option value="">انتخاب کارمند…</option>
                {employees.map((emp) => (
                  <option key={emp.id} value={emp.id}>
                    {emp.full_name} ({emp.personnel_no})
                  </option>
                ))}
              </select>
              <span className="mt-1 block text-xs font-normal text-slate-500 dark:text-slate-400">
                فقط پرونده‌های کارمندی؛ بدون این پیوند، حساب نمی‌تواند خسارت ثبت کند.
              </span>
            </Field>
          )}
          <div className="sm:col-span-2">
            <button
              type="submit"
              disabled={submitting}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {submitting ? 'در حال ایجاد…' : 'ایجاد کاربر'}
            </button>
          </div>
        </form>
      </Card>

      <Card className="!p-0">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-slate-800">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">همهٔ کاربران</h2>
        </div>
        {error && (
          <div className="p-5">
            <ErrorBanner message={error} />
          </div>
        )}
        {loading ? (
          <Spinner />
        ) : (
          <div className="scroll-x">
            <table className="w-full text-start text-sm">
              <thead className="border-b border-slate-100 text-xs uppercase tracking-wide text-slate-400 dark:border-slate-800 dark:text-slate-500">
                <tr>
                  <th className="px-5 py-2 font-medium">نام کاربری</th>
                  <th className="px-5 py-2 font-medium">نام و نام خانوادگی</th>
                  <th className="px-5 py-2 font-medium">نقش</th>
                  <th className="px-5 py-2 font-medium">کارمند مرتبط</th>
                  <th className="px-5 py-2 font-medium">وضعیت</th>
                  <th className="px-5 py-2 font-medium" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {users.map((user) => {
                  const linked = user.employee_id
                    ? employees.find((e) => e.id === user.employee_id)
                    : undefined
                  return (
                  <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="px-5 py-3 font-medium text-slate-900 dark:text-slate-100">{user.username}</td>
                    <td className="px-5 py-3">{user.full_name}</td>
                    <td className="px-5 py-3">
                      {user.role === 'admin' ? (
                        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200">
                          {ROLE_LABELS.admin}
                        </span>
                      ) : (
                        <select
                          value={user.role}
                          onChange={(e) => handleRoleChange(user, e.target.value as Role)}
                          className="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
                        >
                          {ASSIGNABLE_ROLES.map((r) => (
                            <option key={r} value={r}>
                              {ROLE_LABELS[r]}
                            </option>
                          ))}
                        </select>
                      )}
                    </td>
                    <td className="px-5 py-3 text-slate-600 dark:text-slate-300">
                      {linked ? (
                        <span>
                          {linked.full_name}{' '}
                          <span className="dir-ltr font-mono text-xs text-slate-400">({linked.personnel_no})</span>
                        </span>
                      ) : user.role === 'employee' ? (
                        <span className="text-amber-700 dark:text-amber-300">بدون پیوند</span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="px-5 py-3">
                      <span
                        className={
                          'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ' +
                          (user.is_active
                            ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300'
                            : 'bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300')
                        }
                      >
                        {user.is_active ? 'فعال' : 'غیرفعال'}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-end">
                      <div className="flex justify-end gap-2">
                        {user.role !== 'admin' && (
                          <button
                            type="button"
                            onClick={() => handleToggleActive(user)}
                            className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                          >
                            {user.is_active ? 'غیرفعال کردن' : 'فعال کردن'}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => openResetPassword(user)}
                          className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                        >
                          بازنشانی گذرواژه
                        </button>
                      </div>
                    </td>
                  </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {resetTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="reset-password-title"
          onClick={closeResetPassword}
        >
          <div
            className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-5 shadow-lg dark:border-slate-700 dark:bg-slate-900"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 id="reset-password-title" className="text-base font-semibold text-slate-900 dark:text-slate-50">
              بازنشانی گذرواژه
            </h2>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              گذرواژهٔ جدید برای{' '}
              <span className="font-medium text-slate-700 dark:text-slate-200">{resetTarget.username}</span>
              {' '}({resetTarget.full_name})
            </p>
            <form onSubmit={handleResetPassword} className="mt-4 space-y-3">
              {resetError && <ErrorBanner message={resetError} />}
              <Field label="گذرواژهٔ جدید">
                <input
                  type="password"
                  value={resetPassword}
                  onChange={(e) => setResetPassword(e.target.value)}
                  className={inputClass}
                  autoComplete="new-password"
                  autoFocus
                  minLength={MIN_PASSWORD_LEN}
                  required
                />
              </Field>
              <Field label="تکرار گذرواژه">
                <input
                  type="password"
                  value={resetConfirm}
                  onChange={(e) => setResetConfirm(e.target.value)}
                  className={inputClass}
                  autoComplete="new-password"
                  minLength={MIN_PASSWORD_LEN}
                  required
                />
              </Field>
              <p className="text-xs text-slate-500 dark:text-slate-400">حداقل {MIN_PASSWORD_LEN} کاراکتر.</p>
              <div className="flex justify-end gap-2 pt-1">
                <button
                  type="button"
                  onClick={closeResetPassword}
                  disabled={resetSubmitting}
                  className="rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50 disabled:opacity-60 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                >
                  انصراف
                </button>
                <button
                  type="submit"
                  disabled={resetSubmitting}
                  className="rounded-lg bg-brand-600 px-3 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {resetSubmitting ? 'در حال ذخیره…' : 'ذخیرهٔ گذرواژه'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
