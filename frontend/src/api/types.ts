// Wire types for the REST API.
//
// These are NOT hand-maintained: every alias below re-exports a schema that is
// GENERATED from backend/api/openapi.yaml into schema.d.ts. The spec
// is the single source of truth — to change a field, edit the spec and run
// `npm run gen:api` (CI fails if the checked-in schema is stale).
//
// The aliases exist so application code keeps importing friendly names
// (`Claim`, `User`, …) instead of `components['schemas']['Claim']`.

import type { components } from './schema'

type S = components['schemas']

// Enumerations
export type Role = S['Role']
export type EmploymentStatus = S['EmploymentStatus']
export type Relation = S['Relation']
export type BeneficiaryType = S['BeneficiaryType']
export type ClaimStatus = S['ClaimStatus']

// Entities
export type User = S['User']
export type Employee = S['Employee']
export type Dependent = S['Dependent']
export type ServiceType = S['ServiceType']
export type InsuranceContract = S['InsuranceContract']
export type CoveragePlan = S['CoveragePlan']
export type CoverageRule = S['CoverageRule']
export type Claim = S['Claim']
export type AuditLog = S['AuditLog']

// Read models
export type RemainingCap = S['RemainingCap']
export type ReportSummary = S['ReportSummary']
export type SpendByEmployee = S['EmployeeSpend']
export type SpendByServiceType = S['ServiceTypeSpend']
export type SpendByMonth = S['MonthSpend']

// Envelopes
export interface Paginated<T> {
  items: T[]
  total: number
}

export type ApiError = S['Error']
