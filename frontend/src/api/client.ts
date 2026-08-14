import axios from 'axios'

import { translateApiError } from '../lib/errorMessages'

type AppConfig = { apiBaseUrl?: string }

function resolveApiBaseUrl(): string {
  const cfg = (window as unknown as { __APP_CONFIG__?: AppConfig }).__APP_CONFIG__
  return cfg?.apiBaseUrl || '/api/v1'
}

// Runtime value from /config.js (Docker: API_BASE_URL env). Falls back to the
// same-origin nginx/Vite proxy path when config is missing.
export const client = axios.create({ baseURL: resolveApiBaseUrl() })

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

/**
 * The user-facing message for any failure. The API answers in English (it is a
 * machine contract, also consumed by the parent system), so translation happens
 * here — see lib/errorMessages.ts.
 */
export function apiErrorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data as { error?: string } | undefined
    if (data?.error) return translateApiError(data.error)
    // No response at all: the API is unreachable or the request timed out.
    if (err.code === 'ERR_NETWORK' || err.code === 'ECONNABORTED') {
      return 'ارتباط با سرور برقرار نشد؛ دوباره تلاش کنید.'
    }
    if (err.message) return translateApiError(err.message)
  }
  if (err instanceof Error) return translateApiError(err.message)
  return 'خطای پیش‌بینی‌نشده.'
}
