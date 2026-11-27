import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'
import type { Period } from '@/shared/lib/period'
import { getPayrollSummary } from '../api/payroll'

export const payrollKey = (userId: string | undefined) => [...privateQueries, userId, 'payroll'] as const

export function usePayroll(period: Period) {
  const { user, refresh } = useSession()
  const query = useQuery({
    queryKey: [...payrollKey(user?.id), period.start_date, period.end_date],
    queryFn: ({ signal }) => getPayrollSummary(period, signal),
    enabled: Boolean(user),
    gcTime: 0,
  })
  const error = query.error
  useEffect(() => {
    if (error instanceof ApiError && error.status === 401) void refresh()
  }, [error, refresh])
  return query
}
