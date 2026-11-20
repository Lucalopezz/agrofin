import { apiRequest } from '@/shared/lib/api-client'

export type Employee = {
  id: string
  name: string
  active: boolean
}

export const employeesApi = {
  list(active: boolean | undefined, signal: AbortSignal) {
    const query = active === undefined ? '' : `?active=${active}`
    return apiRequest<Employee[]>(`/employees${query}`, { signal })
  },

  create(name: string) {
    return apiRequest<Employee>('/employees', {
      method: 'POST',
      body: JSON.stringify({ name }),
    })
  },

  rename(id: string, name: string) {
    return apiRequest<Employee>(`/employees/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ name }),
    })
  },

  deactivate(id: string) {
    return apiRequest<void>(`/employees/${id}`, { method: 'DELETE' })
  },

  reactivate(id: string) {
    return apiRequest<void>(`/employees/${id}/reactivate`, { method: 'PATCH' })
  },
}
