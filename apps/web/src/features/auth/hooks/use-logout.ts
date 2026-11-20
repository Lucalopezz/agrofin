import { useMutation, useQueryClient } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { logout } from '../api/logout'
import { sessionQuery } from './use-session-query'

export function useLogout() {
  const client = useQueryClient()
  const { mutateAsync } = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      await client.cancelQueries({ queryKey: sessionQuery.queryKey })
      client.setQueryData(sessionQuery.queryKey, null)
      client.removeQueries({ queryKey: privateQueries })
    },
  })
  return mutateAsync
}
