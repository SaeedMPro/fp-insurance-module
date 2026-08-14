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
    setSubmitting(true)
    try {
      await createUser({
        username,
        password,
        full_name: fullName,
        role,
        employee_id: role === 'employee' && employeeId ? employeeId : null,
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
    // A select fires onChange immediately — confirm before changing an account's
    // access level so a stray click can't silently reassign roles.
    if (!window.confirm(`نقش «${user.username}» به «${ROLE_LABELS[newRole]}» تغییر کند؟`)) {
      reload() // snap the select back to the stored value
      return
    }
    try {
      await updateUser(user.id, { role: newRole })
      showToast(`نقش ${user.username} به‌روزرسانی شد.`, 'success')
      reload()
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    }
  }

  async function handleToggleActive(user: User) {
    try {
      await updateUser(user.id, { is_active: !user.is_active })
      showToast(`کاربر ${user.username} ${user.is_active ? 'غیرفعال شد' : 'فعال شد'}.`, 'success')
      reload()
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    }
  }

  async function handleResetPassword(user: User) {
    const next = window.prompt(`گذرواژهٔ جدید برای ${user.username} را وارد کنید:`)
    if (!next) return
    try {
      await updateUser(user.id, { password: next })
      showToast(`گذرواژهٔ ${user.username} بازنشانی شد.`, 'success')
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-50">کاربران</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          مدیریت حساب‌های سامانه و نقش‌های آن‌ها. حساب‌های با نقش کارمند باید به یک پروندهٔ کارمندی
          متصل شوند. مدیر سامانه فقط از طریق seed یا دستور <span className="dir-ltr font-mono text-xs">make create-admin</span> ساخته می‌شود.
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
              <select value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} className={inputClass}>
                <option value="">هیچ‌کدام</option>
                {employees.map((emp) => (
                  <option key={emp.id} value={emp.id}>
                    {emp.full_name} ({emp.personnel_no})
                  </option>
                ))}
              </select>
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
                  <th className="px-5 py-2 font-medium">وضعیت</th>
                  <th className="px-5 py-2 font-medium" />
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                {users.map((user) => (
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
                        <button
                          type="button"
                          onClick={() => handleToggleActive(user)}
                          className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                        >
                          {user.is_active ? 'غیرفعال کردن' : 'فعال کردن'}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleResetPassword(user)}
                          className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
                        >
                          بازنشانی گذرواژه
                        </button>
                      </div>
                    </td>
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
