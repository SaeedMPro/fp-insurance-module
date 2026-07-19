import { client } from './client'
import type { Dependent, Employee, Paginated, RemainingCap } from './types'

export interface ListEmployeesParams {
  q?: string
  page?: number
  page_size?: number
}

export async function listEmployees(params: ListEmployeesParams = {}): Promise<Paginated<Employee>> {
  const { data } = await client.get<Paginated<Employee>>('/employees', { params })
  return data
}

export interface CreateEmployeeRequest {
  personnel_no: string
  full_name: string
  national_id: string
  hire_date: string
  department: string
  plan_id: string | null
}

export async function createEmployee(req: CreateEmployeeRequest): Promise<Employee> {
  const { data } = await client.post<Employee>('/employees', req)
  return data
}

export async function getEmployee(id: string): Promise<Employee> {
  const { data } = await client.get<Employee>(`/employees/${id}`)
  return data
}

export interface UpdateEmployeeRequest {
  employment_status?: string
  plan_id?: string | null
  department?: string
  full_name?: string
}

export async function updateEmployee(id: string, req: UpdateEmployeeRequest): Promise<Employee> {
  const { data } = await client.patch<Employee>(`/employees/${id}`, req)
  return data
}

export async function listDependents(employeeId: string): Promise<Dependent[]> {
  const { data } = await client.get<Dependent[]>(`/employees/${employeeId}/dependents`)
  return data
}

export interface CreateDependentRequest {
  full_name: string
  relation: 'spouse' | 'child' | 'parent'
  national_id: string
  birth_date: string
}

export async function createDependent(employeeId: string, req: CreateDependentRequest): Promise<Dependent> {
  const { data } = await client.post<Dependent>(`/employees/${employeeId}/dependents`, req)
  return data
}

export async function getRemainingCaps(employeeId: string): Promise<RemainingCap[]> {
  const { data } = await client.get<RemainingCap[]>(`/employees/${employeeId}/remaining-caps`)
  return data
}
