import { useCallback, useEffect, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useLogin } from './hooks/use-login'
import { useLogout } from './hooks/use-logout'
import { useSessionQuery } from './hooks/use-session-query'
import { SessionContext, type Session } from './session-context'

export function SessionProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient()
  const query = useSessionQuery()
  const login = useLogin()
  const logout = useLogout()
  const { refetch } = query

  const refresh = useCallback(async () => {
    await refetch({ cancelRefetch: false })
  }, [refetch])

  useEffect(() => {
    // Uma sessão expirada também descarta os dados privados já consultados.
    if (query.data === null) client.removeQueries({ queryKey: privateQueries })
  }, [client, query.data])

  let session: Session
  if (query.isPending || (query.isError && query.isFetching)) {
    session = { status: 'loading', user: null }
  } else if (query.isError) {
    session = { status: 'error', user: null }
  } else if (query.data) {
    session = { status: 'authenticated', user: query.data }
  } else {
    session = { status: 'anonymous', user: null }
  }

  return (
    <SessionContext.Provider value={{ ...session, refresh, login, logout }}>
      {children}
    </SessionContext.Provider>
  )
}
