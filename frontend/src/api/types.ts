// Wire types mirroring backend/internal/models/models.go json tags exactly.
// Do not rename fields -- these are the API contract.

export type Role = 'admin' | 'reviewer' | 'employee' | 'auditor'

export type EmploymentStatus = 'active' | 'terminated'

export type Relation = 'self' | 'spouse' | 'child' | 'parent'

export type BeneficiaryType = 'self' | 'dependent'

export type ClaimStatus =
  | 'draft'
  | 'submitted'
  | 'under_review'
  | 'returned_for_docs'
  | 'approved'
  | 'rejected'
  | 'payment_calculated'
  | 'paid'
  | 'closed'

export interface User {
  id: string
  username: string
  full_name: string
  role: Role
  employee_id: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Employee {
  id: string
  personnel_no: string
  full_name: string
  national_id: string
  employment_status: EmploymentStatus
  hire_date: string
  department: string
  plan_id: string | null
  created_at: string
  updated_at: string
}

export interface Dependent {
  id: string
  employee_id: string
  full_name: string
  relation: Relation
  national_id: string
  birth_date: string | null
  created_at: string
}

export interface ServiceType {
  id: string
  code: string
  name: string
  name_fa: string
  created_at: string
}

export interface InsuranceContract {
  id: string
  name: string
  start_date: string
  end_date: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CoveragePlan {
  id: string
  contract_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface CoverageRule {
  id: string
  plan_id: string
  service_type_id: string
  coverage_percent: number
  per_claim_cap: number | null
  annual_cap: number | null
  waiting_period_days: number
  eligible_relations: Relation[]
  effective_from: string
  effective_to: string | null
  created_by: string | null
  created_at: string
}

export interface Claim {
  id: string
  employee_id: string
  beneficiary_type: BeneficiaryType
  dependent_id: string | null
  service_type_id: string
  plan_id: string
  requested_amount: number
  receipt_date: string
  description: string
  status: ClaimStatus
  coverage_percent_applied: number | null
  payable_amount: number | null
  rejection_reason?: string
  submitted_at: string | null
  reviewed_by: string | null
  reviewed_at: string | null
  paid_at: string | null
  closed_at: string | null
  created_by: string
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: string
  entity_type: string
  entity_id: string
  action: string
  actor_user_id: string | null
  actor_username: string
  before_data: Record<string, unknown> | null
  after_data: Record<string, unknown> | null
  metadata: Record<string, unknown> | null
  occurred_at: string
}

export interface RemainingCap {
  service_type_code: string
  service_type_name: string
  coverage_percent: number
  per_claim_cap: number | null
  annual_cap: number | null
  used_annual: number
  remaining_annual: number
}

export interface ReportSummary {
  total_claims: number
  total_paid_amount: number
  pending_review: number
  approved_awaiting_payment: number
  rejected: number
}

export interface SpendByEmployee {
  employee_id: string
  employee_name: string
  personnel_no: string
  total_paid: number
  claim_count: number
}

export interface SpendByServiceType {
  service_type_code: string
  service_type_name: string
  total_paid: number
  claim_count: number
}

export interface SpendByMonth {
  month: string
  total_paid: number
  claim_count: number
}

export interface Paginated<T> {
  items: T[]
  total: number
}

export interface ApiError {
  error: string
}
