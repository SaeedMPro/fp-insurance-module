import { client } from './client'
import type { ReportSummary, SpendByEmployee, SpendByMonth, SpendByServiceType } from './types'

export interface ReportDateRange {
  from?: string
  to?: string
}

export async function getSummary(params: ReportDateRange = {}): Promise<ReportSummary> {
  const { data } = await client.get<ReportSummary>('/reports/summary', { params })
  return data
}

export async function getSpendByEmployee(params: ReportDateRange = {}): Promise<SpendByEmployee[]> {
  const { data } = await client.get<SpendByEmployee[]>('/reports/spend-by-employee', { params })
  return data
}

export async function getSpendByServiceType(params: ReportDateRange = {}): Promise<SpendByServiceType[]> {
  const { data } = await client.get<SpendByServiceType[]>('/reports/spend-by-service-type', { params })
  return data
}

export async function getSpendByMonth(params: ReportDateRange = {}): Promise<SpendByMonth[]> {
  const { data } = await client.get<SpendByMonth[]>('/reports/spend-by-month', { params })
  return data
}
