import { useEffect, useRef, useState, type ReactNode } from 'react'
import { SessionContext, type Session } from './session-context'
import { authApi, type Credentials } from '@/features/auth/api/auth-api'
import { ApiError } from '@/shared/lib/api-client'

export function SessionProvider({ children }: { children: ReactNode }) {
  const requestVersion = useRef(0)
  const [session, setSession] = useState<Session>({ status: 'loading', user: null })

  async function refresh() {
    const version = ++requestVersion.current
    setSession({ status: 'loading', user: null })
    try {
      const { user } = await authApi.me()
      if (version === requestVersion.current)
        setSession({ status: 'authenticated', user })
    } catch (error) {
      if (version === requestVersion.current)
        setSession({
          status:
            error instanceof ApiError && error.status === 401 ? 'anonymous' : 'error',
          user: null,
        })
    }
  }

  // A versão impede que uma consulta antiga sobrescreva um login ou logout posterior.
  useEffect(() => {
    const version = ++requestVersion.current
    let cancelled = false
    // A resposta só vale enquanto não houver outra operação de sessão mais recente.
    void authApi
      .me()
      .then(({ user }) => {
        if (!cancelled && version === requestVersion.current)
          setSession({ status: 'authenticated', user })
      })
      .catch((error: unknown) => {
        if (!cancelled && version === requestVersion.current)
          setSession({
            status:
              error instanceof ApiError && error.status === 401 ? 'anonymous' : 'error',
            user: null,
          })
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function login(input: Credentials) {
    ++requestVersion.current
    await authApi.login(input)
    // O login cria o cookie, mas somente /me retorna o perfil do cliente.
    const { user } = await authApi.me()
    setSession({ status: 'authenticated', user })
  }

  async function logout() {
    ++requestVersion.current
    await authApi.logout()
    setSession({ status: 'anonymous', user: null })
  }

  return (
    <SessionContext.Provider value={{ ...session, refresh, login, logout }}>
      {children}
    </SessionContext.Provider>
  )
}
