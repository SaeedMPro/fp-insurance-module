import type { Role } from '../api/types'

/**
 * The single source of truth for "which role sees what".
 *
 * Both the router's guards (app/router.tsx) and the sidebar (components/Layout)
 * are generated from this table, so a role change can no longer be applied to
 * one and forgotten in the other. It intentionally mirrors — and must stay
 * consistent with — the backend's RBAC in internal/transport/http/router.go,
 * which is the actual enforcement point; this table only decides what the UI
 * offers.
 */
export interface RouteDef {
  /** Router path. */
  path: string
  /** Roles allowed to open it. */
  roles: Role[]
  /** Sidebar label (Persian); omit to keep the route out of the nav. */
  nav?: string
  /** Marks the route as the landing page for these roles. */
  homeFor?: Role[]
}

export const ROUTES = {
  claims: {
    path: '/claims',
    roles: ['employee', 'reviewer', 'admin', 'auditor'],
    homeFor: ['employee', 'reviewer', 'admin'],
  },
  newClaim: { path: '/claims/new', roles: ['employee', 'admin'], nav: 'ثبت درخواست جدید' },
  claimDetail: { path: '/claims/:id', roles: ['employee', 'reviewer', 'admin', 'auditor'] },
  myCoverage: { path: '/my-coverage', roles: ['employee'], nav: 'پوشش بیمه‌ای من' },
  employees: { path: '/employees', roles: ['admin', 'reviewer'], nav: 'کارکنان' },
  newEmployee: { path: '/employees/new', roles: ['admin'] },
  employeeDetail: { path: '/employees/:id', roles: ['admin', 'reviewer'] },
  coverageRules: { path: '/coverage-rules', roles: ['admin'], nav: 'قوانین پوشش' },
  contracts: { path: '/contracts', roles: ['admin'], nav: 'قراردادها' },
  plans: { path: '/plans', roles: ['admin'], nav: 'طرح‌های پوشش' },
  users: { path: '/users', roles: ['admin'], nav: 'کاربران' },
  reports: { path: '/reports', roles: ['admin', 'auditor'], nav: 'گزارش‌ها', homeFor: ['auditor'] },
  auditLogs: { path: '/audit-logs', roles: ['admin', 'auditor'], nav: 'تاریخچه اقدامات' },
} as const satisfies Record<string, RouteDef>

export type RouteKey = keyof typeof ROUTES

/**
 * The claims list is one screen with a role-dependent title, so its nav label
 * cannot live in the table above.
 */
const CLAIMS_NAV_LABEL: Partial<Record<Role, string>> = {
  employee: 'درخواست‌های من',
  reviewer: 'کارتابل بررسی',
  admin: 'همه درخواست‌ها',
  auditor: 'همه درخواست‌ها',
}

export interface NavItem {
  to: string
  label: string
}

/** Sidebar items for a role, in table order. */
export function navFor(role: Role): NavItem[] {
  const items: NavItem[] = []
  const claimsLabel = CLAIMS_NAV_LABEL[role]
  if (claimsLabel && ROUTES.claims.roles.includes(role)) {
    items.push({ to: ROUTES.claims.path, label: claimsLabel })
  }
  for (const def of Object.values(ROUTES) as RouteDef[]) {
    if (def.path === ROUTES.claims.path) continue
    if (def.nav && def.roles.includes(role)) {
      items.push({ to: def.path, label: def.nav })
    }
  }
  return items
}

/** Landing route for a role after login. */
export function homeFor(role: Role): string {
  for (const def of Object.values(ROUTES) as RouteDef[]) {
    if (def.homeFor?.includes(role)) return def.path
  }
  return ROUTES.claims.path
}
