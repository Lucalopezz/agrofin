import { apiRequest } from '@/shared/lib/api-client'
import type { Employee } from '../types'

export function listEmployees(active: boolean | undefined, signal: AbortSignal) {
  const query = active === undefined ? '' : `?active=${active}`
  return apiRequest<Employee[]>(`/employees${query}`, { signal })
}
