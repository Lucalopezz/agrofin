import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import { listExpenses } from '../api/expenses'

export const expensesKey = (userId: string | undefined) => [...privateQueries, userId, 'expenses'] as const

export function useExpenses(period: Period) {
  const { user, refresh } = useSession()
  const query = useQuery({
    queryKey: [...expensesKey(user?.id), period.start_date, period.end_date],
    queryFn: ({ signal }) => listExpenses(period, signal),
    enabled: Boolean(user),
    gcTime: 0,
  })
  useEffect(() => {
    if (query.error instanceof ApiError && query.error.status === 401) void refresh()
  }, [query.error, refresh])
  return query
}
