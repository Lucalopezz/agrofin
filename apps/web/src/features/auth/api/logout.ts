import { apiRequest } from '@/shared/lib/api-client'

export function logout() {
  return apiRequest<void>('/auth/logout', { method: 'POST' })
}
