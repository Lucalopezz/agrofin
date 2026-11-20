import { apiRequest } from '@/shared/lib/api-client'
import type { Employee } from '../types'

export function renameEmployee(input: { id: string; name: string }) {
  return apiRequest<Employee>(`/employees/${input.id}`, {
    method: 'PATCH',
    body: JSON.stringify({ name: input.name }),
  })
}
