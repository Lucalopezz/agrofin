import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSession } from '@/features/auth/session-context'
import { ApiError } from '@/shared/lib/api-client'
import { employeesApi } from '../api/employees-api'
import { employeeListsKey } from './use-employees'

type EmployeeChange =
  | { action: 'create'; name: string }
  | { action: 'rename'; id: string; name: string }
  | { action: 'deactivate' | 'reactivate'; id: string }

async function saveEmployee(change: EmployeeChange) {
  switch (change.action) {
    case 'create':
      await employeesApi.create(change.name)
      return 'Funcionário cadastrado.'
    case 'rename':
      await employeesApi.rename(change.id, change.name)
      return 'Nome atualizado.'
    case 'deactivate':
      await employeesApi.deactivate(change.id)
      return 'Funcionário desativado.'
    case 'reactivate':
      await employeesApi.reactivate(change.id)
      return 'Funcionário reativado.'
  }
}

export function useEmployeeMutation() {
  const client = useQueryClient()
  const { user, refresh } = useSession()

  return useMutation({
    mutationFn: saveEmployee,
    onSuccess: async () => {
      const filters = { queryKey: employeeListsKey(user?.id) }
      // Uma leitura iniciada antes da escrita pode devolver o estado antigo.
      await client.cancelQueries(filters)
      await client.invalidateQueries(filters)
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 401) void refresh()
    },
  })
}
