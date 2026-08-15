import { createContext, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { login as apiLogin, me } from '../api/auth'
import { clearAuth, getStoredUser, getToken, setAuth, setUnauthorizedHandler } from '../api/client'
import type { User } from '../api/types'

interface AuthContextValue {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  loading: boolean
  login: (username: string, password: string) => Promise<User>
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const storedToken = getToken()
    const storedUser = getStoredUser<User>()
    if (!storedToken || !storedUser) {
      setLoading(false)
      return
    }
    // Re-validate against the API: a token from a previous DB seed is still a
    // valid JWT but its user id no longer exists — force re-login.
    setToken(storedToken)
    setUser(storedUser)
    me()
      .then((fresh) => {
        if (cancelled) return
        setAuth(storedToken, fresh)
        setUser(fresh)
      })
      .catch(() => {
        if (cancelled) return
        clearAuth()
        setToken(null)
        setUser(null)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const logout = useCallback(() => {
    clearAuth()
    setToken(null)
    setUser(null)
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(() => {
      setToken(null)
      setUser(null)
    })
    return () => setUnauthorizedHandler(null)
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const res = await apiLogin({ username, password })
    setAuth(res.token, res.user)
    setToken(res.token)
    setUser(res.user)
    return res.user
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      token,
      isAuthenticated: !!token && !!user,
      loading,
      login,
      logout,
    }),
    [user, token, loading, login, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
