// Reference / config data: service types, contracts, plans, coverage rules.
import { client } from './client'
import type { CoveragePlan, CoverageRule, InsuranceContract, ServiceType } from './types'

export async function listServiceTypes(): Promise<ServiceType[]> {
  const { data } = await client.get<ServiceType[]>('/service-types')
  return data
}

export interface CreateServiceTypeRequest {
  code: string
  name: string
}

export async function createServiceType(req: CreateServiceTypeRequest): Promise<ServiceType> {
  const { data } = await client.post<ServiceType>('/service-types', req)
  return data
}

export async function listContracts(): Promise<InsuranceContract[]> {
  const { data } = await client.get<InsuranceContract[]>('/contracts')
  return data
}

export interface CreateContractRequest {
  name: string
  start_date: string
  end_date: string
  is_active: boolean
}

export async function createContract(req: CreateContractRequest): Promise<InsuranceContract> {
  const { data } = await client.post<InsuranceContract>('/contracts', req)
  return data
}

export async function listPlans(contractId?: string): Promise<CoveragePlan[]> {
  const { data } = await client.get<CoveragePlan[]>('/plans', {
    params: contractId ? { contract_id: contractId } : undefined,
  })
  return data
}

export interface CreatePlanRequest {
  contract_id: string
  name: string
  description: string
}

export async function createPlan(req: CreatePlanRequest): Promise<CoveragePlan> {
  const { data } = await client.post<CoveragePlan>('/plans', req)
  return data
}

export interface ListCoverageRulesParams {
  plan_id?: string
  service_type_id?: string
}

export async function listCoverageRules(params: ListCoverageRulesParams = {}): Promise<CoverageRule[]> {
  const { data } = await client.get<CoverageRule[]>('/coverage-rules', { params })
  return data
}

export interface CreateCoverageRuleRequest {
  plan_id: string
  service_type_id: string
  coverage_percent: number
  per_claim_cap: number | null
  annual_cap: number | null
  waiting_period_days: number
  eligible_relations: string[]
  effective_from: string
}

export async function createCoverageRule(req: CreateCoverageRuleRequest): Promise<CoverageRule> {
  const { data } = await client.post<CoverageRule>('/coverage-rules', req)
  return data
}
