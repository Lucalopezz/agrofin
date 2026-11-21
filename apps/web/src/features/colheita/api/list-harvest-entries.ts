import { apiRequest } from '@/shared/lib/api-client'
import type { HarvestEntry, Period } from '../types'

export function listHarvestEntries(period: Period, signal: AbortSignal) {
  return apiRequest<HarvestEntry[]>(`/harvest-entries?${new URLSearchParams(period)}`, { signal })
}
