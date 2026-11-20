import { useCallback, useEffect, type ReactNode } from 'react'
import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { authApi, type Credentials } from '@/features/auth/api/auth-api'
import { ApiError } from '@/shared/lib/api-client'
import { SessionContext, type Session } from './session-context'

const sessionQuery = queryOptions({
  queryKey: ['session'],
  // O login e o logout atualizam a sessão. Um 401 de rota protegida também
  // força uma nova consulta, mesmo sem mudar o estado de foco da janela.
  staleTime: Infinity,
  queryFn: async ({ signal }) => {
    try {
      const { user } = await authApi.me(signal)
      return user
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) return null
      throw error
    }
  },
})

export function SessionProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient()
  const query = useQuery(sessionQuery)
  const { refetch } = query

  const refresh = useCallback(async () => {
    await refetch({ cancelRefetch: false })
  }, [refetch])

  const { mutateAsync: signIn, reset: resetSignIn } = useMutation({
    gcTime: 0,
    mutationFn: async (input: Credentials) => {
      await authApi.login(input)
      // A troca de cookie invalida leituras iniciadas com a sessão anterior.
      await client.cancelQueries({ queryKey: sessionQuery.queryKey })
      client.removeQueries({ queryKey: privateQueries })
      client.setQueryData(sessionQuery.queryKey, null)
      const user = await client.fetchQuery({ ...sessionQuery, staleTime: 0 })
      if (!user) throw new ApiError(401, 'Não foi possível confirmar a sessão.')
    },
  })

  async function login(input: Credentials) {
    try {
      await signIn(input)
    } finally {
      // As variáveis da mutação contêm a senha e saem do cache ao concluir.
      resetSignIn()
    }
  }

  const { mutateAsync: logout } = useMutation({
    mutationFn: authApi.logout,
    onSuccess: async () => {
      await client.cancelQueries({ queryKey: sessionQuery.queryKey })
      client.setQueryData(sessionQuery.queryKey, null)
      client.removeQueries({ queryKey: privateQueries })
    },
  })

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
