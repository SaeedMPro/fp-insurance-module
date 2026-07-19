import axios from 'axios'

const baseURL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080/api/v1'

export const client = axios.create({ baseURL })

const TOKEN_KEY = 'auth_token'
const USER_KEY = 'auth_user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setAuth(token: string, user: unknown) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

export function getStoredUser<T>(): T | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

client.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  return config
})

// Listeners the AuthProvider registers to react to a forced logout (401).
type Unauthorized401Listener = () => void
let onUnauthorized: Unauthorized401Listener | null = null

export function setUnauthorizedHandler(fn: Unauthorized401Listener | null) {
  onUnauthorized = fn
}

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      clearAuth()
      onUnauthorized?.()
    }
    return Promise.reject(error)
  },
)

export function apiErrorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (err.message) return err.message
  }
  if (err instanceof Error) return err.message
  return 'Unexpected error'
}
