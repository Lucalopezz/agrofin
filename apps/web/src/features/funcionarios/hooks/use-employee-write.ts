import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { ApiError } from '@/shared/lib/api-client'
import { employeeListsKey } from './use-employees'

export function useEmployeeWrite<TInput, TResult>(mutationFn: (input: TInput) => Promise<TResult>) {
  const client = useQueryClient()
  const { user, refresh } = useSession()

  return useMutation({
    mutationFn,
    gcTime: 0,
    onSuccess: async () => {
      const filters = { queryKey: employeeListsKey(user?.id) }
      // Uma leitura iniciada antes da escrita pode devolver o estado antigo.
      const reports = { queryKey: reportsKey(user?.id) }
      await Promise.all([client.cancelQueries(filters), client.cancelQueries(reports)])
      await Promise.all([client.invalidateQueries(filters), client.invalidateQueries(reports)])
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) void refresh()
    },
  })
}
