import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import { sentenceCase } from '@/features/painel/lib/sentence-case'
import { ApiError } from '@/shared/lib/api-client'
import { employeesApi } from '../api/employees-api'

export const employeeListsKey = (userId: string | undefined) =>
  [...privateQueries, userId, 'employees'] as const

export function useEmployees(active?: boolean) {
  const { user, refresh } = useSession()
  const query = useQuery({
    // Cada conta e filtro tem sua própria lista, inclusive no seletor da colheita.
    queryKey: [...employeeListsKey(user?.id), active ?? null],
    queryFn: ({ signal }) => employeesApi.list(active, signal),
    enabled: Boolean(user),
  })
  const { error } = query

  useEffect(() => {
    if (error instanceof ApiError && error.status === 401) void refresh()
  }, [error, refresh])

  return {
    employees: query.data ?? [],
    loading: query.isFetching || query.isPending,
    error: query.isError
      ? error instanceof ApiError
        ? sentenceCase(error.message)
        : 'Não foi possível carregar os funcionários. Confira a conexão.'
      : null,
    reload: () => void query.refetch(),
  }
}
