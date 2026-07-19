import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { ToastProvider } from './context/ToastContext'
import { Layout } from './components/Layout'
import { RequireAuth, RequireRole } from './components/RouteGuards'

import { Login } from './pages/Login'
import { Home } from './pages/Home'
import { NotFound } from './pages/NotFound'
import { ClaimsList } from './pages/claims/ClaimsList'
import { NewClaim } from './pages/claims/NewClaim'
import { ClaimDetail } from './pages/claims/ClaimDetail'
import { MyCoverage } from './pages/employee/MyCoverage'
import { EmployeesList } from './pages/employees/EmployeesList'
import { NewEmployee } from './pages/employees/NewEmployee'
import { EmployeeDetail } from './pages/employees/EmployeeDetail'
import { CoverageRules } from './pages/admin/CoverageRules'
import { Contracts } from './pages/admin/Contracts'
import { Plans } from './pages/admin/Plans'
import { Users } from './pages/admin/Users'
import { Reports } from './pages/auditor/Reports'
import { AuditLog } from './pages/auditor/AuditLog'

function App() {
  return (
    <AuthProvider>
      <ToastProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />

            <Route
              element={
                <RequireAuth>
                  <Layout />
                </RequireAuth>
              }
            >
              <Route path="/" element={<Home />} />

              {/* Claims: employees see their own, reviewers/admins see all */}
              <Route
                path="/claims"
                element={
                  <RequireRole roles={['employee', 'reviewer', 'admin']}>
                    <ClaimsList />
                  </RequireRole>
                }
              />
              <Route
                path="/claims/new"
                element={
                  <RequireRole roles={['employee', 'admin']}>
                    <NewClaim />
                  </RequireRole>
                }
              />
              <Route
                path="/claims/:id"
                element={
                  <RequireRole roles={['employee', 'reviewer', 'admin', 'auditor']}>
                    <ClaimDetail />
                  </RequireRole>
                }
              />

              {/* Employee self-service */}
              <Route
                path="/my-coverage"
                element={
                  <RequireRole roles={['employee']}>
                    <MyCoverage />
                  </RequireRole>
                }
              />

              {/* Employee administration (admin can mutate; reviewer read-only list/detail) */}
              <Route
                path="/employees"
                element={
                  <RequireRole roles={['admin', 'reviewer']}>
                    <EmployeesList />
                  </RequireRole>
                }
              />
              <Route
                path="/employees/new"
                element={
                  <RequireRole roles={['admin']}>
                    <NewEmployee />
                  </RequireRole>
                }
              />
              <Route
                path="/employees/:id"
                element={
                  <RequireRole roles={['admin', 'reviewer']}>
                    <EmployeeDetail />
                  </RequireRole>
                }
              />

              {/* Admin: config-driven policy + reference data + users */}
              <Route
                path="/coverage-rules"
                element={
                  <RequireRole roles={['admin']}>
                    <CoverageRules />
                  </RequireRole>
                }
              />
              <Route
                path="/contracts"
                element={
                  <RequireRole roles={['admin']}>
                    <Contracts />
                  </RequireRole>
                }
              />
              <Route
                path="/plans"
                element={
                  <RequireRole roles={['admin']}>
                    <Plans />
                  </RequireRole>
                }
              />
              <Route
                path="/users"
                element={
                  <RequireRole roles={['admin']}>
                    <Users />
                  </RequireRole>
                }
              />

              {/* Reporting & audit: admin + auditor */}
              <Route
                path="/reports"
                element={
                  <RequireRole roles={['admin', 'auditor']}>
                    <Reports />
                  </RequireRole>
                }
              />
              <Route
                path="/audit-logs"
                element={
                  <RequireRole roles={['admin', 'auditor']}>
                    <AuditLog />
                  </RequireRole>
                }
              />

              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ToastProvider>
    </AuthProvider>
  )
}

export default App
