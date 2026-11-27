import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { ApiError } from '@/shared/lib/api-client'
import { createReceipt, createSale } from '../api/sales'
import type { SaleInput } from '../types'
import { salesKey } from './use-sales'

type Write =
  | { kind: 'sale'; input: SaleInput }
  | { kind: 'receipt'; saleId: string; cents: number }

export function useSaleWrite() {
  const { user, refresh } = useSession()
  const client = useQueryClient()
  const lock = useRef(false)
  const sales = { queryKey: salesKey(user?.id) }
  const reports = { queryKey: reportsKey(user?.id) }
  const mutation = useMutation({
    mutationFn: async (action: Write) => action.kind === 'sale'
      ? { kind: 'sale' as const, saved: await createSale(action.input) }
      : { kind: 'receipt' as const, saved: await createReceipt(action.saleId, action.cents) },
    gcTime: 0,
    retry: false,
    onSuccess: async () => {
      // Venda e recebimento alimentam indicadores por datas distintas.
      await Promise.all([client.cancelQueries(sales), client.cancelQueries(reports)])
      await Promise.all([client.invalidateQueries(sales), client.invalidateQueries(reports)])
    },
    onError: async (error, action) => {
      if (!(error instanceof ApiError)) return
      if (error.status === 401) void refresh()
      if (action.kind === 'receipt' && error.status === 409) {
        // Atualiza o saldo oficial sem reenviar o valor digitado.
        await client.cancelQueries(sales)
        await client.invalidateQueries(sales)
      }
    },
  })

  async function submit(action: Write) {
    // A API grava um registro por requisição aceita. Bloqueia o segundo
    // clique imediatamente, antes de o estado pendente chegar à interface.
    if (lock.current) return undefined
    lock.current = true
    try {
      return await mutation.mutateAsync(action)
    } catch {
      return undefined
    } finally {
      lock.current = false
    }
  }

  function errorFor(kind: Write['kind'], saleId?: string) {
    const { error, variables } = mutation
    if (!error || variables?.kind !== kind || ('saleId' in variables && variables.saleId !== saleId)) return null
    if (error instanceof ApiError && error.status < 500) return error.message
    return 'Não foi possível confirmar o lançamento. Atualize as vendas antes de tentar novamente.'
  }

  return { busy: mutation.isPending, reset: mutation.reset, submit, errorFor }
}
