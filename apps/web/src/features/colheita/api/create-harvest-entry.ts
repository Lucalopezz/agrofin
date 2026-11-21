import { apiRequest } from '@/shared/lib/api-client'
import type { HarvestEntry, HarvestInput } from '../types'

export function createHarvestEntry(input: HarvestInput) {
  return apiRequest<HarvestEntry>('/harvest-entries', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
