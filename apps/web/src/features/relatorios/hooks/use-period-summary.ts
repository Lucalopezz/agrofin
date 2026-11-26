import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import type { Period } from '@/features/colheita/types'
import { ApiError } from '@/shared/lib/api-client'
import { getPeriodDetails, getPeriodSummary } from '../api/period-report'
import type { DetailResource } from '../types'

export const reportsKey = (userId: string | undefined) => [...privateQueries, userId, 'reports'] as const

function useReportQuery<T>(
  resource: DetailResource | 'summary',
  period: Period,
  read: (signal: AbortSignal) => Promise<T>,
) {
  const { user, refresh } = useSession()
  const query = useQuery({
    queryKey: [...reportsKey(user?.id), resource, period.start_date, period.end_date],
    queryFn: ({ signal }) => read(signal),
    enabled: Boolean(user),
    // O relatório guarda dados privados somente enquanto há uma tela observando a consulta.
    gcTime: 0,
  })
  useEffect(() => {
    if (query.error instanceof ApiError && query.error.status === 401) void refresh()
  }, [query.error, refresh])
  return query
}

export function usePeriodSummary(period: Period) {
  return useReportQuery('summary', period, (signal) => getPeriodSummary(period, signal))
}

export function useReportDetails<R extends DetailResource>(resource: R, period: Period) {
  return useReportQuery(resource, period, (signal) => getPeriodDetails(resource, period, signal))
}
