import { apiRequest } from '@/shared/lib/api-client'
import type { Registration } from '../types'

export function register(input: Registration) {
  return apiRequest<{ registered: boolean }>('/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
