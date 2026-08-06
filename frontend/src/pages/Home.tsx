import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'
import { homeFor } from '../app/routes'

// Landing page per role comes from the shared route table (app/routes.ts).
export function Home() {
  const { user } = useAuth()
  if (!user) return <Navigate to="/login" replace />
  return <Navigate to={homeFor(user.role)} replace />
}
