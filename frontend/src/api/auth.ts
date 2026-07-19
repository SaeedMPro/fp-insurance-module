import { client } from './client'
import type { User } from './types'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const { data } = await client.post<LoginResponse>('/auth/login', req)
  return data
}

export async function me(): Promise<User> {
  const { data } = await client.get<User>('/auth/me')
  return data
}
