import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { privateQueries } from '@/app/query-client'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'

export type HarvestResource = 'rates' | 'entries'
export const harvestKey = (userId: string | undefined, resource: HarvestResource) =>
  [...privateQueries, userId, 'harvest', resource] as const

export function useHarvestQuery<T>(
  resource: HarvestResource,
  parameters: string[],
  read: (signal: AbortSignal) => Promise<T>,
  enabled = true,
) {
  const { user, refresh } = useSession()
  const query = useQuery({
    queryKey: [...harvestKey(user?.id, resource), ...parameters],
    queryFn: ({ signal }) => read(signal),
    enabled: Boolean(user) && enabled,
    // Mantém valores financeiros somente em memória e descarta ao sair da tela.
    staleTime: 30_000,
    gcTime: 0,
  })
  useEffect(() => {
    if (query.error instanceof ApiError && query.error.status === 401) void refresh()
  }, [query.error, refresh])
  return query
}
