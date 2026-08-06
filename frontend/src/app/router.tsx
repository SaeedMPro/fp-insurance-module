import { lazy, Suspense } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router-dom'

import { Layout } from '../components/Layout'
import { RequireAuth, RequireRole } from '../components/RouteGuards'
import { Spinner } from '../components/Spinner'
import { Login } from '../pages/Login'
import { Home } from '../pages/Home'
import { NotFound } from '../pages/NotFound'
import { ROUTES } from './routes'
import type { RouteDef } from './routes'

// Route-level code splitting: recharts (the reports bundle) and the admin
// screens no longer ship to an employee who never opens them. Login/Home/
// NotFound stay eager — they are tiny and on the critical path.
const ClaimsList = lazy(() => import('../pages/claims/ClaimsList').then((m) => ({ default: m.ClaimsList })))
const NewClaim = lazy(() => import('../pages/claims/NewClaim').then((m) => ({ default: m.NewClaim })))
const ClaimDetail = lazy(() => import('../pages/claims/ClaimDetail').then((m) => ({ default: m.ClaimDetail })))
const MyCoverage = lazy(() => import('../pages/employee/MyCoverage').then((m) => ({ default: m.MyCoverage })))
const EmployeesList = lazy(() => import('../pages/employees/EmployeesList').then((m) => ({ default: m.EmployeesList })))
const NewEmployee = lazy(() => import('../pages/employees/NewEmployee').then((m) => ({ default: m.NewEmployee })))
const EmployeeDetail = lazy(() => import('../pages/employees/EmployeeDetail').then((m) => ({ default: m.EmployeeDetail })))
const CoverageRules = lazy(() => import('../pages/admin/CoverageRules').then((m) => ({ default: m.CoverageRules })))
const Contracts = lazy(() => import('../pages/admin/Contracts').then((m) => ({ default: m.Contracts })))
const Plans = lazy(() => import('../pages/admin/Plans').then((m) => ({ default: m.Plans })))
const Users = lazy(() => import('../pages/admin/Users').then((m) => ({ default: m.Users })))
const Reports = lazy(() => import('../pages/auditor/Reports').then((m) => ({ default: m.Reports })))
const AuditLog = lazy(() => import('../pages/auditor/AuditLog').then((m) => ({ default: m.AuditLog })))

// Each guarded route pairs a definition from routes.ts with its element, so the
// allowed roles are never restated here.
const GUARDED: Array<{ def: RouteDef; element: React.ReactNode }> = [
  { def: ROUTES.claims, element: <ClaimsList /> },
  { def: ROUTES.newClaim, element: <NewClaim /> },
  { def: ROUTES.claimDetail, element: <ClaimDetail /> },
  { def: ROUTES.myCoverage, element: <MyCoverage /> },
  { def: ROUTES.employees, element: <EmployeesList /> },
  { def: ROUTES.newEmployee, element: <NewEmployee /> },
  { def: ROUTES.employeeDetail, element: <EmployeeDetail /> },
  { def: ROUTES.coverageRules, element: <CoverageRules /> },
  { def: ROUTES.contracts, element: <Contracts /> },
  { def: ROUTES.plans, element: <Plans /> },
  { def: ROUTES.users, element: <Users /> },
  { def: ROUTES.reports, element: <Reports /> },
  { def: ROUTES.auditLogs, element: <AuditLog /> },
]

export function AppRouter() {
  return (
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

          {GUARDED.map(({ def, element }) => (
            <Route
              key={def.path}
              path={def.path}
              element={
                <RequireRole roles={[...def.roles]}>
                  <Suspense fallback={<Spinner />}>{element}</Suspense>
                </RequireRole>
              }
            />
          ))}

          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
