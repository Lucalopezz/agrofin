import { apiRequest } from '@/shared/lib/api-client'
import type { Credentials } from '../types'

export function login(input: Credentials) {
  return apiRequest<{ authenticated: boolean }>('/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
