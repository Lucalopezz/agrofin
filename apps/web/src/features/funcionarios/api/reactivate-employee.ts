import { apiRequest } from '@/shared/lib/api-client'

export function reactivateEmployee(id: string) {
  return apiRequest<void>(`/employees/${id}/reactivate`, { method: 'PATCH' })
}
