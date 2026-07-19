import { client } from './client'
import type { Role, User } from './types'

export async function listUsers(): Promise<User[]> {
  const { data } = await client.get<User[]>('/admin/users')
  return data
}

export interface CreateUserRequest {
  username: string
  password: string
  full_name: string
  role: Role
  employee_id: string | null
}

export async function createUser(req: CreateUserRequest): Promise<User> {
  const { data } = await client.post<User>('/admin/users', req)
  return data
}

export interface UpdateUserRequest {
  role?: Role
  is_active?: boolean
  password?: string
}

export async function updateUser(id: string, req: UpdateUserRequest): Promise<User> {
  const { data } = await client.patch<User>(`/admin/users/${id}`, req)
  return data
}
