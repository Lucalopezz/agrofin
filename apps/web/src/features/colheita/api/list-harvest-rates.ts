import { apiRequest } from '@/shared/lib/api-client'
import type { Rate } from '../types'

export function listHarvestRates(signal: AbortSignal) {
  return apiRequest<Rate[]>('/harvest-rates', { signal })
}
