import { apiRequest } from '@/shared/lib/api-client'

export function deactivateEmployee(id: string) {
  return apiRequest<void>(`/employees/${id}`, { method: 'DELETE' })
}
