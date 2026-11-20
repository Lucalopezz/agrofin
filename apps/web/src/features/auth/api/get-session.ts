import { apiRequest } from '@/shared/lib/api-client'
import type { User } from '../types'

export function getSession(signal: AbortSignal) {
  return apiRequest<{ authenticated: boolean; user: User }>('/auth/me', { signal })
}
