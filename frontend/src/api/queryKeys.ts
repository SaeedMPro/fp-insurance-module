import type { ListClaimsParams } from './claims'
import type { ListAuditLogsParams } from './auditLogs'
import type { ListEmployeesParams } from './employees'
import type { ReportDateRange } from './reports'

/**
 * Central query-key factory for react-query.
 *
 * Keys live in one place so a mutation can invalidate exactly the lists it
 * affects without guessing at key shapes (and without the "invalidate
 * everything" hammer). Keys are hierarchical: qk.claims() is a prefix of
 * qk.claimsList(params), so invalidating the former refreshes every claims
 * query regardless of its filters.
 */
export const qk = {
  claims: () => ['claims'] as const,
  claimsList: (params: ListClaimsParams) => ['claims', 'list', params] as const,
  claim: (id: string) => ['claims', 'detail', id] as const,
  claimHistory: (id: string) => ['claims', 'history', id] as const,

  employees: () => ['employees'] as const,
  employeesList: (params: ListEmployeesParams) => ['employees', 'list', params] as const,
  employee: (id: string) => ['employees', 'detail', id] as const,
  dependents: (employeeId: string) => ['employees', 'dependents', employeeId] as const,
  remainingCaps: (employeeId: string) => ['employees', 'remaining-caps', employeeId] as const,

  serviceTypes: () => ['service-types'] as const,
  contracts: () => ['contracts'] as const,
  plans: (contractId?: string) => (contractId ? (['plans', contractId] as const) : (['plans'] as const)),
  coverageRules: () => ['coverage-rules'] as const,

  users: () => ['users'] as const,

  auditLogs: (params: ListAuditLogsParams) => ['audit-logs', params] as const,

  reports: () => ['reports'] as const,
  reportSummary: (range: ReportDateRange) => ['reports', 'summary', range] as const,
  reportByEmployee: (range: ReportDateRange) => ['reports', 'by-employee', range] as const,
  reportByServiceType: (range: ReportDateRange) => ['reports', 'by-service-type', range] as const,
  reportByMonth: (range: ReportDateRange) => ['reports', 'by-month', range] as const,
}
