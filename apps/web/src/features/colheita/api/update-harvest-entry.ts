import { apiRequest } from '@/shared/lib/api-client'
import type { HarvestEntry, HarvestItemInput } from '../types'

export function updateHarvestEntry(input: { id: string; items: HarvestItemInput[] }) {
  return apiRequest<HarvestEntry>(`/harvest-entries/${input.id}`, {
    method: 'PATCH',
    body: JSON.stringify({ items: input.items }),
  })
}
