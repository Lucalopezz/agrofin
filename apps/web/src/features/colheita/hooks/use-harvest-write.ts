import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
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
      await client.cancelQueries(filters)
      await client.invalidateQueries(filters)
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
