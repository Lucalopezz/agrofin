import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { employeeListsKey } from '@/features/funcionarios/hooks/use-employees'
import { reportsKey } from '@/features/relatorios/hooks/use-period-summary'
import { ApiError } from '@/shared/lib/api-client'
import { registerDailyWork, registerPayment } from '../api/payroll'
import type { DailyWorkInput, PaymentInput } from '../types'
import { payrollKey } from './use-payroll'

type Write =
  | { kind: 'payment'; input: PaymentInput }
  | { kind: 'dailyWork'; input: DailyWorkInput }

export function usePayrollWrite() {
  const { user, refresh } = useSession()
  const client = useQueryClient()
  const lock = useRef(false)
  const payroll = { queryKey: payrollKey(user?.id) }
  const reports = { queryKey: reportsKey(user?.id) }
  const mutation = useMutation({
    mutationFn: async (action: Write) => action.kind === 'payment'
      ? { kind: 'payment' as const, saved: await registerPayment(action.input) }
      : { kind: 'dailyWork' as const, saved: await registerDailyWork(action.input) },
    gcTime: 0,
    retry: false,
    onSuccess: async () => {
      await Promise.all([client.cancelQueries(payroll), client.cancelQueries(reports)])
      await Promise.all([client.invalidateQueries(payroll), client.invalidateQueries(reports)])
    },
    onError: async (error, action) => {
      if (!(error instanceof ApiError)) return
      if (error.status === 401) void refresh()
      if (action.kind === 'payment' && error.status === 409) {
        // Outra baixa pode ter alterado o saldo. Consulta o valor oficial
        // e deixa o rascunho para o cliente conferir e decidir sobre novo envio.
        await client.cancelQueries(payroll)
        await client.invalidateQueries(payroll)
      }
      if (action.kind === 'dailyWork' && (error.status === 409 || error.status === 404)) {
        await client.invalidateQueries({ queryKey: employeeListsKey(user?.id) })
      }
    },
  })

  async function submit(action: Write) {
    // Cada requisição aceita gera um lançamento. O ref cobre dois eventos
    // recebidos antes de o React desativar o botão.
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

  function errorFor(kind: Write['kind']) {
    const { error, variables } = mutation
    if (!error || variables?.kind !== kind) return null
    if (error instanceof ApiError && error.status < 500) return error.message
    return 'Não foi possível confirmar o lançamento. Confira o resumo antes de tentar novamente.'
  }

  return { busy: mutation.isPending, reset: mutation.reset, submit, errorFor }
}
