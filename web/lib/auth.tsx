'use client'

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react'
import { useRouter, usePathname } from 'next/navigation'

interface User {
  id: string
  email: string
  username: string
  role: 'admin' | 'user'
  status: string
}

interface AuthContextType {
  user: User | null
  loading: boolean
  isAdmin: boolean
  login: (email: string, password: string) => Promise<void>
  register: (email: string, username: string, password: string) => Promise<void>
  logout: () => void
  getAccessToken: () => Promise<string | null>
}

const AuthContext = createContext<AuthContextType | null>(null)

const PUBLIC_PATHS = ['/login', '/register']

let accessToken: string | null = null

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const router = useRouter()
  const pathname = usePathname()

  const clearAuth = useCallback(() => {
    accessToken = null
    localStorage.removeItem('refresh_token')
    setUser(null)
  }, [])

  const refreshAccessToken = useCallback(async (): Promise<string | null> => {
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) return null

    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      if (!res.ok) {
        clearAuth()
        return null
      }
      const data = await res.json()
      accessToken = data.access_token
      return accessToken
    } catch {
      clearAuth()
      return null
    }
  }, [clearAuth])

  const getAccessToken = useCallback(async (): Promise<string | null> => {
    if (accessToken) return accessToken
    return refreshAccessToken()
  }, [refreshAccessToken])

  const fetchMe = useCallback(async (token: string) => {
    try {
      const res = await fetch('/api/v1/auth/me', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (res.ok) {
        const userData = await res.json()
        setUser(userData)
        return true
      }
    } catch {}
    return false
  }, [])

  // 初始化：尝试用 refresh token 恢复会话
  useEffect(() => {
    const init = async () => {
      const token = await refreshAccessToken()
      if (token) {
        await fetchMe(token)
      }
      setLoading(false)
    }
    init()
  }, [refreshAccessToken, fetchMe])

  // 路由守卫
  useEffect(() => {
    if (loading) return
    if (!user && !PUBLIC_PATHS.includes(pathname)) {
      router.replace('/login')
    }
    if (user && PUBLIC_PATHS.includes(pathname)) {
      router.replace('/')
    }
  }, [user, loading, pathname, router])

  const login = async (email: string, password: string) => {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'login failed')
    }
    const data = await res.json()
    accessToken = data.access_token
    localStorage.setItem('refresh_token', data.refresh_token)
    setUser(data.user)
    router.replace('/')
  }

  const register = async (email: string, username: string, password: string) => {
    const res = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, username, password }),
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'registration failed')
    }
    const data = await res.json()
    accessToken = data.access_token
    localStorage.setItem('refresh_token', data.refresh_token)
    setUser(data.user)
    router.replace('/')
  }

  const logout = () => {
    clearAuth()
    router.replace('/login')
  }

  return (
    <AuthContext.Provider value={{
      user,
      loading,
      isAdmin: user?.role === 'admin',
      login,
      register,
      logout,
      getAccessToken,
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

// authFetch: 自动附加 Authorization header 的 fetch 包装
export async function authFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const token = accessToken
  const headers = new Headers(options.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return fetch(url, { ...options, headers })
}
