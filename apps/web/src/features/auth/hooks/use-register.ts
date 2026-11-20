import { useMutation } from '@tanstack/react-query'
import { register } from '../api/register'
import type { Registration } from '../types'

export function useRegister() {
  const { mutateAsync, reset } = useMutation({ mutationFn: register, gcTime: 0 })

  return async (input: Registration) => {
    try {
      await mutateAsync(input)
    } finally {
      // Descarta as credenciais que estavam nas variáveis do cadastro.
      reset()
    }
  }
}
