import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import { listSales } from '../api/sales'

export const salesKey = (userId: string | undefined) => [...privateQueries, userId, 'sales'] as const

export function useSales(period: Period) {
  const { user, refresh } = useSession()
  const query = useQuery({
    queryKey: [...salesKey(user?.id), period.start_date, period.end_date],
    queryFn: ({ signal }) => listSales(period, signal),
    enabled: Boolean(user),
    gcTime: 0,
  })
  const { error } = query
  useEffect(() => {
    if (error instanceof ApiError && error.status === 401) void refresh()
  }, [error, refresh])
  return query
}
