import { apiRequest } from '@/shared/lib/api-client'
import type { Employee } from '../types'

export function createEmployee(name: string) {
  return apiRequest<Employee>('/employees', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}
