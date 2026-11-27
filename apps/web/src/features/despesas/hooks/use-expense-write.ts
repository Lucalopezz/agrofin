import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { ApiError } from '@/shared/lib/api-client'
import { createExpense, deleteExpense, updateExpense } from '../api/expenses'
import type { ExpenseInput } from '../types'
import { expensesKey } from './use-expenses'

type ExpenseWrite =
  | { kind: 'create'; input: ExpenseInput }
  | { kind: 'update'; id: string; input: ExpenseInput }
  | { kind: 'delete'; id: string }

export function useExpenseWrite() {
  const { user, refresh } = useSession()
  const client = useQueryClient()
  const lock = useRef(false)
  const mutation = useMutation({
    gcTime: 0,
    retry: false,
    mutationFn: async (action: ExpenseWrite) => {
      switch (action.kind) {
        case 'create': await createExpense(action.input); break
        case 'update': await updateExpense(action.id, action.input); break
        case 'delete': await deleteExpense(action.id); break
      }
    },
    onSuccess: async () => {
      // Uma edição pode mover a despesa de um período para outro. Invalida
      // também o caixa e os detalhes dos relatórios da mesma conta.
      const lists = { queryKey: expensesKey(user?.id) }
      const reports = { queryKey: reportsKey(user?.id) }
      await Promise.all([client.cancelQueries(lists), client.cancelQueries(reports)])
      await Promise.all([client.invalidateQueries(lists), client.invalidateQueries(reports)])
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) void refresh()
    },
  })

  async function submit(action: ExpenseWrite) {
    // O cadastro grava uma despesa por envio. O ref bloqueia cliques seguidos
    // antes de o React atualizar o estado pendente dos botões.
    if (lock.current) return false
    lock.current = true
    try {
      await mutation.mutateAsync(action)
      return true
    } catch {
      return false
    } finally {
      lock.current = false
    }
  }

  function errorFor(kind: ExpenseWrite['kind'], id?: string) {
    const action = mutation.variables
    const { error } = mutation
    if (!error || action?.kind !== kind || ('id' in action && action.id !== id)) return null
    if (error instanceof ApiError && error.status === 404) {
      return 'Despesa indisponível. Atualize a lista para conferir os registros do período.'
    }
    if (error instanceof ApiError && error.status < 500) return error.message
    return 'Não foi possível confirmar a alteração. Atualize a lista antes de tentar novamente.'
  }

  return { busy: mutation.isPending, reset: mutation.reset, submit, errorFor }
}
