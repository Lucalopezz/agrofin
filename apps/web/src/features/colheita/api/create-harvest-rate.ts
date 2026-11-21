import { apiRequest } from '@/shared/lib/api-client'
import type { Rate, RateInput } from '../types'

export function createHarvestRate(input: RateInput) {
  return apiRequest<Rate>('/harvest-rates', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
