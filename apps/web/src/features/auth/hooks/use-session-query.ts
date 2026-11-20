import { queryOptions, useQuery } from '@tanstack/react-query'
import { ApiError } from '@/shared/lib/api-client'
import { getSession } from '../api/get-session'

export const sessionQuery = queryOptions({
  queryKey: ['session'],
  // Login, logout e um 401 de rota protegida controlam a atualização da sessão.
  staleTime: Infinity,
  queryFn: async ({ signal }) => {
    try {
      const { user } = await getSession(signal)
      return user
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) return null
      throw error
    }
  },
})

export function useSessionQuery() {
  return useQuery(sessionQuery)
}
