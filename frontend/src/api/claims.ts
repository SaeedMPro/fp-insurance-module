import { client } from './client'
import type { AuditLog, BeneficiaryType, Claim, ClaimStatus, Paginated } from './types'

export interface ListClaimsParams {
  status?: ClaimStatus
  employee_id?: string
  service_type_id?: string
  from?: string
  to?: string
  page?: number
  page_size?: number
}

export async function listClaims(params: ListClaimsParams = {}): Promise<Paginated<Claim>> {
  const { data } = await client.get<Paginated<Claim>>('/claims', { params })
  return data
}

export interface CreateClaimRequest {
  employee_id: string
  beneficiary_type: BeneficiaryType
  dependent_id: string | null
  service_type_id: string
  requested_amount: number
  receipt_date: string
  description: string
}

export async function createClaim(req: CreateClaimRequest): Promise<Claim> {
  const { data } = await client.post<Claim>('/claims', req)
  return data
}

export async function getClaim(id: string): Promise<Claim> {
  const { data } = await client.get<Claim>(`/claims/${id}`)
  return data
}

export async function getClaimHistory(id: string): Promise<AuditLog[]> {
  const { data } = await client.get<AuditLog[]>(`/claims/${id}/history`)
  return data
}

async function transition(id: string, action: string, body?: unknown): Promise<Claim> {
  const { data } = await client.post<Claim>(`/claims/${id}/${action}`, body)
  return data
}

export const submitClaim = (id: string) => transition(id, 'submit')
export const resubmitClaim = (id: string) => transition(id, 'resubmit')
export const startReview = (id: string) => transition(id, 'start-review')
export const approveClaim = (id: string) => transition(id, 'approve')
export const rejectClaim = (id: string, reason: string) => transition(id, 'reject', { reason })
export const returnForDocs = (id: string, reason: string) => transition(id, 'return-for-docs', { reason })
export const markPaid = (id: string) => transition(id, 'mark-paid')
export const closeClaim = (id: string) => transition(id, 'close')
