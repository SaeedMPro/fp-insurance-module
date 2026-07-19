import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'

const DEFAULT_ROUTE: Record<string, string> = {
  employee: '/claims',
  reviewer: '/claims',
  admin: '/claims',
  auditor: '/reports',
}

export function Home() {
  const { user } = useAuth()
  if (!user) return <Navigate to="/login" replace />
  return <Navigate to={DEFAULT_ROUTE[user.role] ?? '/login'} replace />
}
