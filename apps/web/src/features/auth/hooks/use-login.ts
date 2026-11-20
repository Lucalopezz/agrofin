import { useMutation, useQueryClient } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { ApiError } from '@/shared/lib/api-client'
import { login } from '../api/login'
import type { Credentials } from '../types'
import { sessionQuery } from './use-session-query'

export function useLogin() {
  const client = useQueryClient()
  const { mutateAsync, reset } = useMutation({
    gcTime: 0,
    mutationFn: async (input: Credentials) => {
      await login(input)
      // A troca de cookie invalida leituras iniciadas com a sessão anterior.
      await client.cancelQueries({ queryKey: sessionQuery.queryKey })
      client.removeQueries({ queryKey: privateQueries })
      client.setQueryData(sessionQuery.queryKey, null)
      const user = await client.fetchQuery({ ...sessionQuery, staleTime: 0 })
      if (!user) throw new ApiError(401, 'Não foi possível confirmar a sessão.')
    },
  })

  return async (input: Credentials) => {
    try {
      await mutateAsync(input)
    } finally {
      // As variáveis da mutação contêm a senha e saem do cache ao concluir.
      reset()
    }
  }
}
