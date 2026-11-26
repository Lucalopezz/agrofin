import { useRef } from 'react'
import { useMutation, useQueryClient, type QueryFilters } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { ApiError } from '@/shared/lib/api-client'
import { harvestKey, type HarvestResource } from './use-harvest-query'

export function useHarvestWrite<TInput, TResult>(
  resource: HarvestResource,
  write: (input: TInput) => Promise<TResult>,
) {
  const { user, refresh } = useSession()
  const client = useQueryClient()
  const locked = useRef(false)
  const mutation = useMutation({
    mutationFn: write,
    gcTime: 0,
    retry: false,
    onSuccess: async () => {
      const filters = { queryKey: harvestKey(user?.id, resource) }
      // Tarifas novas não mudam a tarifa fotografada nos lançamentos existentes.
      const affected: QueryFilters[] = resource === 'entries' ? [filters, { queryKey: reportsKey(user?.id) }] : [filters]
      await Promise.all(affected.map((filter) => client.cancelQueries(filter)))
      await Promise.all(affected.map((filter) => client.invalidateQueries(filter)))
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) void refresh()
    },
  })

  async function submit(input: TInput) {
    // As escritas não têm chave de idempotência. Bloqueia inclusive dois eventos no mesmo render.
    if (locked.current) return
    locked.current = true
    try {
      return await mutation.mutateAsync(input)
    } catch {
      // O formulário exibe o erro da mutação e conserva o rascunho.
      return undefined
    } finally {
      locked.current = false
    }
  }

  return { ...mutation, submit }
}
